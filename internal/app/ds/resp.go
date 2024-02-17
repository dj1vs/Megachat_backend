package ds

import "time"

// payloads

type FrontRespPayload struct {
	Status  string
	Message string
}

type CodingRespPayload struct {
	Status      string
	Data        []byte
	Segment_num int32
	Segment_cnt int32
}

// responses

type FrontResp struct {
	Username string
	Time     time.Time
	Payload  FrontRespPayload
}

type CodingResp struct {
	Username string
	Time     time.Time
	Payload  CodingRespPayload
}
