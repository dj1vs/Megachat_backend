package app

import (
	"context"
	"log"
	"megachat/internal/app/config"
	"net/http"
	"strconv"
	"time"

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

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	UUID uuid.UUID

	// The websocket connection.
	Conn *websocket.Conn

	// Buffered channel of outbound messages.
	Send chan []byte
}
type Application struct {
	config *config.Config

	server *http.Server

	// Registered clients.
	Clients map[*Client]bool

	// Inbound messages from the clients.
	Broadcast chan []byte

	// Register requests from the clients.
	Register chan *Client

	// Unregister requests from clients.
	Unregister chan *Client

	kp *KafkaPayload

	numberToUUID map[int64]uuid.UUID
}

func New(ctx context.Context) (*Application, error) {
	cfg, err := config.NewConfig(ctx)
	if err != nil {
		return nil, err
	}

	a := &Application{
		config:       cfg,
		Broadcast:    make(chan []byte),
		Register:     make(chan *Client),
		Unregister:   make(chan *Client),
		Clients:      make(map[*Client]bool),
		numberToUUID: make(map[int64]uuid.UUID),
		kp: &KafkaPayload{
			Slices:      make(map[int64][][]byte, 0),
			Segments:    make(map[int64]int64, 0),
			LastUpdated: make(map[int64]time.Time, 0),
			SliceSender: make(map[int64]string, 0),
		},
	}

	go a.ListenForRecentKafkaMessages()

	return a, nil
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

	return byte_segments

}
