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

	"github.com/IBM/sarama"
)

func (a *Application) SendToCoding(frontReq *ds.FrontReq) error {
	byte_segments := a.TextToByteSegments(frontReq.Payload.Data)

	segments_cnt := len(byte_segments)
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
	}

	return nil
}

func (a *Application) ServeCoding(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "text/plain")

	method := r.Method

	if method != http.MethodPost {
		//fmt.Fprint(w, "You should send POST request")
		http.Error(w, "Method not allowed", http.StatusBadRequest)
		// w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	var requestBody ds.CodingResp

	err = json.Unmarshal(body, &requestBody)
	if err != nil {
		log.Println("Невозможно распознать сообщение от сервиса кодирования:")
		log.Println(err)
		return
	}

	// SEND JSON TO KAFKA

	sarama_config := sarama.NewConfig()
	sarama_config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer([]string{a.config.KafkaHost + ":" + strconv.Itoa(a.config.KafkaPort)}, sarama_config)
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer producer.Close()

	kafka_request := &sarama.ProducerMessage{
		Topic: "megachat",
		Key:   sarama.StringEncoder(strconv.FormatInt(requestBody.Time, 10)),
		Value: sarama.ByteEncoder(body),
	}

	_, _, err = producer.SendMessage(kafka_request)
	if err != nil {
		log.Printf("Failed to send message to mr. Kafka: %v", err)
	}

	a.SendRespToFront(&ds.FrontResp{
		Username: requestBody.Username,
		Time:     requestBody.Time,
		Payload: ds.FrontRespPayload{
			Status:  "ok",
			Message: "",
		},
	})
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
