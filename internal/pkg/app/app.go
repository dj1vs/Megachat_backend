package app

import (
	"context"
	"log"
	"megachat/internal/app/ws"
	"net/http"
	"time"
)

type Application struct {
}

func New(ctx context.Context) (*Application, error) {
	return &Application{}, nil
}

func (a *Application) StartServer() {
	log.Println("Server started")

	hub := ws.NewHub()
	go hub.Run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.ServeWs(hub, w, r)
	})

	server := &http.Server{
		Addr:              "127.0.0.1:8800",
		ReadHeaderTimeout: 5 * time.Second,
	}

	err := server.ListenAndServe()
	if err != nil {
		log.Fatal("Listen and serve:", err)
	}

	log.Println("Server is down")
}
