package app

import (
	"encoding/json"
	"log"
	"megachat/internal/app/ds"
	"strconv"
	"time"

	"github.com/IBM/sarama"
)

type KafkaPayload struct {
	Slices      map[int64][][]byte
	Segments    map[int64]int64
	LastUpdated map[int64]time.Time
	SliceSender map[int64]string
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
		case <-time.After(a.config.KafkaTimeout):
			continue
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

	a.kp.SliceSender[sliceID] = msg.Username

	a.kp.LastUpdated[sliceID] = time.Now()
}

func (a *Application) ProcessNewSliceSegment(msg *ds.CodingResp) error {
	sliceID := msg.Time
	segCount := msg.Payload.Segment_cnt
	segNum := msg.Payload.Segment_num
	segData := msg.Payload.Data
	segStatus := msg.Payload.Status

	if segStatus != "ok" {
		err := a.SendKafkaSliceError(sliceID, a.kp.SliceSender[sliceID])
		a.DeleteSlice(sliceID)

		return err
	}

	// save segment
	a.kp.Slices[sliceID][segNum] = segData

	// update segments data
	a.kp.Segments[sliceID]++

	a.kp.LastUpdated[sliceID] = time.Now()

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
	msg := &ds.FrontMsg{
		Username: username,
		Time:     sliceID,
	}

	msgData := make([]byte, 0)
	for _, seg := range a.kp.Slices[sliceID] {
		msgData = append(msgData, seg...)
	}

	msg.Payload = ds.FrontMsgPayload{
		Status:  "ok",
		Message: "",
		Data:    string(msgData),
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

func (a *Application) SendKafkaSliceLost(sliceID int64, username string) error {
	msg := &ds.FrontMsg{
		Username: username,
		Time:     sliceID,
	}

	msg.Payload = ds.FrontMsgPayload{
		Status:  "error",
		Message: "Часть сообщения потеряна",
		Data:    "",
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

func (a *Application) SendKafkaSliceError(sliceID int64, username string) error {
	msg := &ds.FrontMsg{
		Username: username,
		Time:     sliceID,
	}

	msg.Payload = ds.FrontMsgPayload{
		Status:  "error",
		Message: "Произошла ошибка при декодировании одного изсегментов сообщения",
		Data:    "",
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

func (a *Application) DeleteSlice(sliceID int64) {
	delete(a.numberToUUID, sliceID)

	delete(a.kp.Slices, sliceID)
	delete(a.kp.LastUpdated, sliceID)
	delete(a.kp.Segments, sliceID)
	delete(a.kp.SliceSender, sliceID)
}

func (a *Application) CheckLostSlices() {
	for sliceID := range a.kp.LastUpdated {
		if a.kp.LastUpdated[sliceID].Add(a.config.KafkaTimeout).Compare(time.Now()) == -1 {
			a.SendKafkaSliceLost(sliceID, a.kp.SliceSender[sliceID])
			a.DeleteSlice(sliceID)
		}
	}
}
