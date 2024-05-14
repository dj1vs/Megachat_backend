package ds

// payloads

type CodingRespPayload struct {
	Status      string `json:"status"`
	Data        []byte `json:"data"`
	Segment_num int32  `json:"segment_num"`
	Segment_cnt int32  `json:"segment_cnt"`
}

// responses

type CodingResp struct {
	Username string            `json:"username"`
	Time     int64             `json:"time"`
	Payload  CodingRespPayload `json:"payload"`
}
