package ds

// payloads

type FrontMsgPayload struct {
	Data string `json:"data"`
}

// messages

type FrontMsg struct {
	Username string          `json:"username"`
	Time     int64           `json:"time"`
	Payload  FrontMsgPayload `json:"payload"`
}
