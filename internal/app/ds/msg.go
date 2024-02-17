package ds

import "time"

// payloads

type FrontMsgPayload struct {
	Status  string
	Message string
	Data    string
}

// messages

type FrontMsg struct {
	Username string
	Time     time.Time
	Payload  FrontMsgPayload
}
