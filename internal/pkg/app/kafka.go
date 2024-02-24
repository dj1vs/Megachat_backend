package app

import (
	"encoding/json"
	"fmt"
	"log"
	"megachat/internal/app/ds"
	"strconv"
	"time"

	"github.com/IBM/sarama"
)

type KafkaPayload struct {
	Slices   map[int64][][]byte
	Segments map[int64]int64
}

func (a *Application) ListenForRecentKafkaMessages() {
	sarama_config := sarama.NewConfig()
	sarama_config.Consumer.Return.Errors = true

	brokers := []string{a.config.KafkaHost + ":" + strconv.Itoa(a.config.KafkaPort)}
	topics := []string{a.config.KafkaTopic}

	consumer, err := sarama.NewConsumer(brokers, sarama_config)
	if err != nil {
		log.Fatalf("Failed to create Kafka consumer: %v", err)
	} else {
		log.Println("Kafka consumer created")
	}
	defer consumer.Close()

	partConsumer, err := consumer.ConsumePartition(topics[0], 0, sarama.OffsetNewest)
	if err != nil {
		log.Fatalf("Failed to consume partition: %v", err)
	} else {
		log.Println("Partition consumed")
	}
	defer partConsumer.Close()

	for {
		select {
		case msg := <-partConsumer.Messages():
			err = a.ProcessKafkaMessage(msg)
			if err != nil {
				log.Println(err)
			}
		case <-time.After(5 * time.Second):
			fmt.Println("No messages received for 5 seconds")
		}
	}
}

func (a *Application) ProcessKafkaMessage(msg *sarama.ConsumerMessage) error {
	var codingResp ds.CodingResp

	err := json.Unmarshal(msg.Value, &codingResp)
	if err != nil {
		return err
	}

	sliceID := codingResp.Time

	_, sliceFound := a.kp.Slices[sliceID]

	if !sliceFound {
		a.ProcessNewKafkaSlice(&codingResp)
	} else {
		a.ProcessNewSliceSegment(&codingResp)
	}

	return nil
}

func (a *Application) ProcessNewKafkaSlice(msg *ds.CodingResp) {
	sliceID := msg.Time
	segCount := msg.Payload.Segment_cnt
	segNum := msg.Payload.Segment_num
	segData := msg.Payload.Data

	// add new slice
	a.kp.Slices[sliceID] = make([][]byte, segCount)

	// initialize segments data
	a.kp.Segments[sliceID] = 1

	// save first segment
	a.kp.Slices[sliceID][segNum] = segData
}

// TODO: check if segments have errors
// TODO: check if some slices are missing
func (a *Application) ProcessNewSliceSegment(msg *ds.CodingResp) error {
	sliceID := msg.Time
	segCount := msg.Payload.Segment_cnt
	segNum := msg.Payload.Segment_num
	segData := msg.Payload.Data

	// save segment
	a.kp.Slices[sliceID][segNum] = segData

	// update segments data
	a.kp.Segments[sliceID]++

	// check if we got all slice segments
	if a.kp.Segments[sliceID] == int64(segCount) {
		err := a.SendKafkaSlice(sliceID, msg.Username)
		if err != nil {
			return err
		}
	}

	return nil
}
func (a *Application) SendKafkaSlice(sliceID int64, username string) error {
	msgPayload := make([]byte, 0)
	for _, seg := range a.kp.Slices[sliceID] {
		msgPayload = append(msgPayload, seg...)
	}

	msg := &ds.FrontMsg{
		Username: username,
		Time:     sliceID,
		Payload: ds.FrontMsgPayload{
			Status:  "ok",
			Message: "",
			Data:    string(msgPayload),
		},
	}

	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	go func() {
		a.Broadcast <- msgJSON
	}()

	return nil
}
