package ds

// messages

type FrontMsg struct {
	Username string `json:"username"`
	Time     int64  `json:"time"`
	Message  string `json:"message"`
	Status   string `json:"status"`
}
