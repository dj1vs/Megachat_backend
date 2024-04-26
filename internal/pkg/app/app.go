package app

// @BasePath /

import (
	"context"
	"log"
	"megachat/internal/app/config"
	"net/http"
	"strconv"
	"time"

	_ "megachat/docs"
	// swagger embed files
	// gin-swagger middleware
)

const (
	// Размер сегмента сообщения в байтах
	segmentByteSize = 140
)

type Application struct {
	config *config.Config

	server *http.Server

	kp *KafkaPayload

	httpClient *http.Client
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
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
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
	// go func() {
	// 	r := gin.Default()
	// 	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	// 	r.Run(":8080")
	// }()

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

	log.Printf("Сообщение успешно разбито на %v сегментов\n", len(byte_segments))
	for segment_num, segment := range byte_segments {
		log.Printf("Сегмент #%v", segment_num)
		log.Println(segment)
	}

	return byte_segments

}
