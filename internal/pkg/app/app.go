package app

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"megachat/internal/app/config"
	"megachat/internal/app/ds"
	"net/http"
	"strconv"
	"time"
)

const (
	// Размер сегмента сообщения в байтах
	segmentByteSize = 140
)

type Application struct {
	config *config.Config

	server *http.Server

	kp *KafkaPayload
}

func New(ctx context.Context) (*Application, error) {
	cfg, err := config.NewConfig(ctx)
	if err != nil {
		return nil, err
	}

	a := &Application{
		config: cfg,
		kp: &KafkaPayload{
			Slices:      make(map[int64][][]byte, 0),
			Segments:    make(map[int64]int64, 0),
			LastUpdated: make(map[int64]time.Time, 0),
			SliceSender: make(map[int64]string, 0),
		},
	}

	go a.ListenForRecentKafkaMessages()

	go func() {
		for {
			a.CheckLostSlices()
			time.Sleep(a.config.KafkaTimeout)
		}
	}()

	return a, nil
}

func (a *Application) StartServer() {
	log.Println("Server started")

	http.HandleFunc("/front", func(w http.ResponseWriter, r *http.Request) {
		a.ServeFront(w, r)
	})

	http.HandleFunc("/coding", func(w http.ResponseWriter, r *http.Request) {
		a.ServeCoding(w, r)
	})

	a.server = &http.Server{
		Addr:              a.config.ServerHost + ":" + strconv.Itoa(a.config.ServerPort),
		ReadHeaderTimeout: 5 * time.Second,
	}

	err := a.server.ListenAndServe()
	if err != nil {
		log.Fatal("Listen and serve:", err)
	}

	log.Println("Server is down")
}

func (a *Application) ServeFront(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	var request ds.FrontReq

	var response *ds.FrontResp

	err = json.Unmarshal(body, &request)

	if err != nil {
		response = &ds.FrontResp{
			Username: "",
			Time:     time.Now().Unix(),
			Payload: ds.FrontRespPayload{
				Status:  "error",
				Message: "Невозможно распознать JSON запрос",
			},
		}
	} else {
		err = a.SendToCoding(&request)

		if err != nil {
			response = &ds.FrontResp{
				Username: request.Username,
				Time:     request.Time,
				Payload: ds.FrontRespPayload{
					Status:  "error",
					Message: "Произошла ошибка при отправка сообщения на сервис кодирования",
				},
			}
		} else {
			response = &ds.FrontResp{
				Username: request.Username,
				Time:     request.Time,
				Payload: ds.FrontRespPayload{
					Status:  "ok",
					Message: "",
				},
			}
		}

	}

	a.SendRespToFront(response)
}

func (a *Application) TextToByteSegments(text string) [][]byte {
	byte_segments := [][]byte{}

	text_bytes := []byte(text)

	for len(text_bytes) > 0 {
		var byte_segment []byte
		isLarge := (len(text_bytes) >= segmentByteSize)
		if isLarge {
			byte_segment = text_bytes[:segmentByteSize]
		} else {
			byte_segment = text_bytes
		}

		byte_segments = append(byte_segments, byte_segment)

		if isLarge {
			text_bytes = text_bytes[segmentByteSize:]
		} else {
			text_bytes = []byte{}
		}
	}

	return byte_segments

}
