package ds

import "time"

// payloads

type FrontRespPayload struct {
	Status  string
	Message string
}

type CodingRespPayload struct {
	Status      string `json:"status"`
	Data        []byte `json:"data"`
	Segment_num int32  `json:"segment_num"`
	Segment_cnt int32  `json:"segment_cnt"`
}

// responses

type FrontResp struct {
	Username string
	Time     time.Time
	Payload  FrontRespPayload
}

type CodingResp struct {
	Username string            `json:"username"`
	Time     time.Time         `json:"time"`
	Payload  CodingRespPayload `json:"payload"`
}
