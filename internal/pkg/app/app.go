package app

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Application struct {
	r *gin.Engine
}

func New(ctx context.Context) (*Application, error) {
	return &Application{}, nil
}

func (a *Application) StartServer() {
	log.Println("Server started")

	a.r = gin.Default()

	a.r.GET("ws/ping", a.ws_ping)

	a.r.Run(":8800")

	log.Println("Server is down")
}

func (a *Application) ws_ping(c *gin.Context) {
	log.Println("ws_ping start")
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// panic(err)
		log.Printf("%s, error while Upgrading websocket connection\n", err.Error())
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	defer conn.Close()
	for {
		mt, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}

		// log.Printf("recv: %s, type: %s", message, websocket.)
		err = conn.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}

}
