package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"megachat/internal/app/ds"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 1024

	// Размер сегмента сообщения в байтах
	segmentByteSize = 140

	codingURL = "http://192.168.207.207:3000/serv/"

	kafkaURL = "172.23.80.195:9092"
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var numberToUUID map[int64]uuid.UUID
var kafkaSlices map[int64][][]byte

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	UUID uuid.UUID

	// The websocket connection.
	Conn *websocket.Conn

	// Buffered channel of outbound messages.
	Send chan []byte
}
type Application struct {
	server *http.Server

	// Registered clients.
	Clients map[*Client]bool

	// Inbound messages from the clients.
	Broadcast chan []byte

	// Register requests from the clients.
	Register chan *Client

	// Unregister requests from clients.
	Unregister chan *Client
}

func New(ctx context.Context) (*Application, error) {

	numberToUUID = make(map[int64]uuid.UUID)

	a := &Application{
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
	}

	kafkaSlices = make(map[int64][][]byte, 0)

	go a.kafkaConsumeRoutine()

	return &Application{
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
	}, nil
}

func (a *Application) kafkaConsumeRoutine() {
	sarama_config := sarama.NewConfig()
	sarama_config.Consumer.Return.Errors = true

	brokers := []string{kafkaURL}
	topics := []string{"megachat"}

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
			var kafkaMsg ds.CodingResp

			err := json.Unmarshal(msg.Value, &kafkaMsg)
			if err != nil {
				log.Println("Can't unmarshal kafka msg")
			}

			_, ok := kafkaSlices[kafkaMsg.Time]
			if !ok {
				kafkaSlices[kafkaMsg.Time] = make([][]byte, kafkaMsg.Payload.Segment_cnt)
			}

			ws_msg := &ds.FrontMsg{
				Username: kafkaMsg.Username,
				Time:     kafkaMsg.Time,
				Payload: ds.FrontMsgPayload{
					Status:  "ok",
					Message: "",
					Data:    string(kafkaMsg.Payload.Data),
				},
			}

			json_ws_msg, err := json.Marshal(ws_msg)
			if err != nil {
				log.Println(err)
			}

			kafkaSlices[kafkaMsg.Time][kafkaMsg.Payload.Segment_num] = kafkaMsg.Payload.Data

			fmt.Println("Received message from Kafka:", string(msg.Value))

			go func() {
				a.Broadcast <- []byte(numberToUUID[kafkaMsg.Time].String() + " " + string(json_ws_msg))
			}()

		case <-time.After(5 * time.Second):
			fmt.Println("No messages received for 5 seconds")
		}
	}
}

func (a *Application) StartServer() {
	log.Println("Server started")

	go a.RunWS()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		a.ServeWs(w, r)
	})

	http.HandleFunc("/coding", func(w http.ResponseWriter, r *http.Request) {
		a.ServeCoding(w, r)
	})

	a.server = &http.Server{
		Addr:              "0.0.0.0:8800",
		ReadHeaderTimeout: 5 * time.Second,
	}

	err := a.server.ListenAndServe()
	if err != nil {
		log.Fatal("Listen and serve:", err)
	}

	log.Println("Server is down")
}

func (a *Application) RunWS() {
	for {
		select {
		case client := <-a.Register:
			a.Clients[client] = true
		case client := <-a.Unregister:
			if _, ok := a.Clients[client]; ok {
				delete(a.Clients, client)
				close(client.Send)
			}
		case message := <-a.Broadcast:
			for client := range a.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(a.Clients, client)
				}
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func (a *Application) ServeWs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	r.Header.Set("Access-Control-Allow-Origin", "*")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{
		UUID: uuid.New(),
		Conn: conn,
		Send: make(chan []byte, 256),
	}
	a.Register <- client

	log.Println("New connection!")
	log.Println("New client uuid: " + client.UUID.String())

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go a.writePump(client)
	go a.readPump(client)
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
		log.Println(err)
		return
	}

	// SEND JSON TO KAFKA

	sarama_config := sarama.NewConfig()
	sarama_config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer([]string{kafkaURL}, sarama_config)
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

	// TODO: send data to kafka

}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from goroutine
func (a *Application) readPump(c *Client) {
	defer func() {
		a.Unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))

		var request ds.FrontReq

		var response *ds.FrontResp

		err = json.Unmarshal(message, &request)

		if err != nil {
			response = &ds.FrontResp{
				Username: "",
				Time:     time.Now().Unix(),
				Payload: ds.FrontRespPayload{
					Status:  "error",
					Message: err.Error(),
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
						Message: err.Error(),
					},
				}
			} else {
				numberToUUID[request.Time] = c.UUID
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

		jsonResponse, err := json.Marshal(response)
		if err != nil {
			fmt.Println("Error: ", err)
		} else {
			a.Broadcast <- []byte(c.UUID.String() + " " + string(jsonResponse))
		}

		// msg_bytes := c.UUID.String() + " " + string(message)

		// c.hub.broadcast <- ([]byte(write_msg))
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (a *Application) writePump(c *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			msg_str := string(message)
			if strings.Split(msg_str, " ")[0] == c.UUID.String() {
				w.Write([]byte(string(message)[strings.Index(string(message), " "):]))
			}

			// Add queued chat messages to the current websocket message.
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

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

		resp, err := http.Post(codingURL, "application/json", bytes.NewBuffer(jsonRequest))
		if err != nil {
			fmt.Println("SendToCoding error sending request: ", err)
			return err
		}
		defer resp.Body.Close()
	}

	return nil
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
