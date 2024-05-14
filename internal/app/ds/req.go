package ds

type CodingReqPayload struct {
	Data        []byte `json:"data"`
	Segment_num int32  `json:"segment_num"`
	Segment_cnt int32  `json:"segment_cnt"`
}

// requests

type FrontReq struct { // TODO: merge with msg
	Username string `json:"username"`
	Time     int64  `json:"time"`
	Message  string `json:"message"`
}

type CodingReq struct {
	Username string           `json:"username"`
	Time     int64            `json:"time"`
	Payload  CodingReqPayload `json:"payload"`
}
