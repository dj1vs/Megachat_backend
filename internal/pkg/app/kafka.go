package app

import (
	"encoding/json"
	"log"
	"megachat/internal/app/ds"
	"strconv"
	"sync"
	"time"

	"github.com/IBM/sarama"
)

type KafkaSliceStatus int

const (
	Success KafkaSliceStatus = iota
	Lost
	Error
)

type KafkaPayload struct {
	Mutex       sync.Mutex
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
			a.kp.Mutex.Lock()
			err = a.ProcessKafkaMessage(msg)
			if err != nil {
				log.Println(err)
			}
			a.kp.Mutex.Unlock()
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

	a.kp.Slices[sliceID][segNum] = segData

	// save segment
	// segStatus := msg.Payload.Status
	// if segStatus == "" {
	//
	// } else {
	// 	a.kp.Slices[sliceID][segNum] = nil
	// }

	// update segments data
	a.kp.Segments[sliceID]++

	a.kp.LastUpdated[sliceID] = time.Now()

	// check if we got all slice segments
	if a.kp.Segments[sliceID] == int64(segCount) {
		isFail := a.SliceHasErrors(sliceID)

		var err error
		if !isFail {
			err = a.SendKafkaSlice(sliceID, Success)
		} else {
			err = a.SendKafkaSlice(sliceID, Error)
		}

		a.DeleteSlice(sliceID)

		if err != nil {
			return err
		}
	}

	return nil
}
func (a *Application) SendKafkaSlice(sliceID int64, status KafkaSliceStatus) error {
	msg := &ds.FrontMsg{
		Username: a.kp.SliceSender[sliceID],
		Time:     sliceID,
	}

	switch status {
	case Success:
		msgData := make([]byte, 0)
		for _, seg := range a.kp.Slices[sliceID] {
			msgData = append(msgData, seg...)
		}

		msg.Payload = ds.FrontMsgPayload{
			Status:  "ok",
			Message: "",
			Data:    string(msgData),
		}
	case Error:
		msg.Payload = ds.FrontMsgPayload{
			Status:  "error",
			Message: "Произошла ошибка при декодировании одного изсегментов сообщения",
			Data:    "",
		}
	case Lost:
		msg.Payload = ds.FrontMsgPayload{
			Status:  "error",
			Message: "Часть сообщения потеряна",
			Data:    "",
		}
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
	delete(a.kp.Slices, sliceID)
	delete(a.kp.LastUpdated, sliceID)
	delete(a.kp.Segments, sliceID)
	delete(a.kp.SliceSender, sliceID)
}

func (a *Application) CheckLostSlices() {
	a.kp.Mutex.Lock()
	defer a.kp.Mutex.Unlock()

	for sliceID := range a.kp.LastUpdated {
		if a.kp.LastUpdated[sliceID].Add(a.config.KafkaTimeout).Compare(time.Now()) == -1 {
			a.SendKafkaSlice(sliceID, Lost)
			a.DeleteSlice(sliceID)
		}
	}
}

func (a *Application) SliceHasErrors(sliceID int64) bool {
	for seg := range a.kp.Slices[sliceID] {

		if a.kp.Slices[sliceID][seg] == nil {
			return true
		}
	}

	return false
}
