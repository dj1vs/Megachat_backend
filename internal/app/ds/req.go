package ds

import "time"

type FrontReqPayload struct {
	data string
}

type FrontReq struct {
	Username string
	time     time.Time
	payload  FrontReqPayload
}
