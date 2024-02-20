package ds

// payloads

type FrontReqPayload struct {
	Data string `json:"data"`
}

type CodingReqPayload struct {
	Data        []byte `json:"data"`
	Segment_num int32  `json:"segment_num"`
	Segment_cnt int32  `json:"segment_cnt"`
}

// requests

type FrontReq struct {
	Username string          `json:"username"`
	Time     int64           `json:"time"`
	Payload  FrontReqPayload `json:"payload"`
}

type CodingReq struct {
	Username string           `json:"username"`
	Time     int64            `json:"time"`
	Payload  CodingReqPayload `json:"payload"`
}
