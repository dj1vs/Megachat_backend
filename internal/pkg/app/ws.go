package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"megachat/internal/app/ds"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

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

	log.Println("New connection! UUID: " + client.UUID.String())

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go a.writePump(client)
	go a.readPump(client)
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
					Message: "Невозможно распознать JSON запрос",
				},
			}
		} else {
			a.numberToUUID[request.Time] = c.UUID
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

		jsonResponse, err := json.Marshal(response)
		if err != nil {
			fmt.Println("Error: ", err)
		} else {
			a.Broadcast <- jsonResponse
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

			var jsonMessage ds.FrontMsg
			err := json.Unmarshal(message, &jsonMessage)
			if err != nil {
				log.Println("Can't unmarshall ws message!")
				continue
			}

			messageTime := jsonMessage.Time

			senderUUID, ok := a.numberToUUID[messageTime]

			if !ok ||
				((jsonMessage.Payload.Data == "") && (senderUUID != c.UUID)) ||
				((jsonMessage.Payload.Data != "") && (senderUUID == c.UUID)) {
				continue
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			w.Write(message)

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
