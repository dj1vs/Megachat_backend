package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"megachat/internal/app/ds"
	"net/http"
	"strconv"
	"time"

	"github.com/IBM/sarama"
)

// @title Прикладной уровень системы обмена сообщениями
// @version 0.0-0

// @host 127.0.0.1:8080
// @schemes http
// @BasePath /

func (a *Application) SendToCoding(frontReq *ds.FrontReq) error {
	byte_segments := a.TextToByteSegments(frontReq.Payload.Data)
	segments_cnt := len(byte_segments)
	log.Printf("SendToCoding: Сообщение успешно разбито на %d сегментов\n", segments_cnt)

	for segment_num, byte_segment := range byte_segments {
		request := &ds.CodingReq{
			Username: frontReq.Username,
			Time:     frontReq.Time,
			Payload: ds.CodingReqPayload{
				Data:        byte_segment,
				Segment_num: int32(segment_num),
				Segment_cnt: int32(segments_cnt),
			},
		}

		jsonRequest, err := json.Marshal(request)
		if err != nil {
			fmt.Println("SendToCoding: не удалось запаковать запрос ", err)
			return err
		}

		condingServiceURL := "http://" + a.config.CodingHost + ":" + strconv.Itoa(a.config.CodingPort) + "/code"

		resp, err := a.httpClient.Post(condingServiceURL, "application/json", bytes.NewBuffer(jsonRequest))
		if err != nil {
			fmt.Println("SendToCoding: не удалось отправить запрос ", err)
			return err
		}
		defer resp.Body.Close()
	}

	return nil
}

// @Summary      Обрабатывает сообщения от сервиса кодирования
// @Accept       json
// @Success      200
// @Failure      400  "Недопустимый метод"  httputil.HTTPError
// @Failure		403 "Ошибка при получении сегмента" httputil.HTTPError
// @Failure	  500  "Ошибка при чтении JSON" httputil.HTTPError
// @Param	message body ds.CodingResp true "Сообщение от сервиса кодирования"
// @Router       /coding [post]
func (a *Application) ServeCoding(w http.ResponseWriter, r *http.Request) {
	log.Println("--> /coding: обработка сообщения от сервера кодирования")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "text/plain")

	method := r.Method

	if method != http.MethodPost {
		log.Println("--> /coding: отправлен неправильный тип запроса")
		http.Error(w, "Method not allowed", http.StatusBadRequest)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("--> /coding: не удалось прочитать тело запроса")
		http.Error(w, "Не удалось прочитать тело запроса", http.StatusInternalServerError)
		return
	}

	var requestBody ds.CodingResp

	err = json.Unmarshal(body, &requestBody)
	if err != nil {
		log.Println("--> /coding: Получено сообщение в неправильном формате:")
		log.Println(err)
		http.Error(w, "Неверный формат сообщения", http.StatusBadRequest)
		return
	}

	log.Println("--> /coding: Сообщение успешно рарспознано, отправка сообщения в Kafka")
	// SEND JSON TO KAFKA
	sarama_config := sarama.NewConfig()
	sarama_config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer([]string{a.config.KafkaHost + ":" + strconv.Itoa(a.config.KafkaPort)}, sarama_config)
	if err != nil {
		log.Fatalf("--> /coding: Не удалось создать Kafka-producer:\n%v", err)
		http.Error(w, "Не удалось поместить сообщение в Kafka", http.StatusInternalServerError)
		return
	}
	defer producer.Close()

	kafka_request := &sarama.ProducerMessage{
		Topic: "megachat",
		Key:   sarama.StringEncoder(strconv.FormatInt(requestBody.Time, 10)),
		Value: sarama.ByteEncoder(body),
	}

	_, _, err = producer.SendMessage(kafka_request)
	if err != nil {
		log.Printf("Не удалось отправить сообщение в Kafka: %v", err)
		http.Error(w, "Не удалось отправить сообщение в Kafka", http.StatusInternalServerError)
	}
}

// @Summary      Обрабатывает сообщения от фронтенда (прикладной уровень)
// @Accept       json
// @Success      200
// @Failure      400  "Недопустимый метод"  httputil.HTTPError
// @Failure	  500  "Ошибка при чтении JSON" httputil.HTTPError
// @Param	message body ds.FrontReq true "Сообщение от фронтенда"
// @Router       /front [post]
func (a *Application) ServeFront(w http.ResponseWriter, r *http.Request) {
	log.Println("--> /front: получено новое сообщение")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	var request ds.FrontReq

	var response *ds.FrontResp

	err = json.Unmarshal(body, &request)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("--> /front: сообщение не удалось распознать")
		response = &ds.FrontResp{
			Username: "",
			Time:     time.Now().Unix(),
			Payload: ds.FrontRespPayload{
				Status:  "error",
				Message: "Невозможно распознать JSON запрос",
			},
		}
	} else {
		log.Println("--> /front: сообщение успешно распознано")
		log.Println(request)
		log.Println("--> /front: отправка сообщения на сервис кодирования")
		err = a.SendToCoding(&request)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println("--> /front: не удалось отправить сообщение сервису кодирования")
			response = &ds.FrontResp{
				Username: request.Username,
				Time:     request.Time,
				Payload: ds.FrontRespPayload{
					Status:  "error",
					Message: "Произошла ошибка при отправка сообщения на сервис кодирования",
				},
			}
		} else {
			w.WriteHeader(http.StatusOK)
			log.Println("--> /front: сообщение успешно отправлено на сервис кодирования")
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

	response_bytes, _ := json.Marshal(response)
	w.Write(response_bytes)

	// a.SendRespToFront(response)
}

func (a *Application) SendRespToFront(msg *ds.FrontResp) error {
	jsonRequest, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("SendToCoding error marshalling request: ", err)
		return err
	}

	condingServiceURL := "http://" + a.config.CodingHost + ":" + strconv.Itoa(a.config.CodingPort) + "/serv/"

	resp, err := http.Post(condingServiceURL, "application/json", bytes.NewBuffer(jsonRequest))
	if err != nil {
		fmt.Println("SendToCoding error sending request: ", err)
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (a *Application) SendMsgToFront(msg *ds.FrontMsg) error {
	jsonRequest, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("SendToCoding error marshalling request: ", err)
		return err
	}

	condingServiceURL := "http://" + a.config.CodingHost + ":" + strconv.Itoa(a.config.CodingPort) + "/serv/"

	resp, err := http.Post(condingServiceURL, "application/json", bytes.NewBuffer(jsonRequest))
	if err != nil {
		fmt.Println("SendToCoding error sending request: ", err)
		return err
	}
	defer resp.Body.Close()

	return nil
}
