package ds

// payloads

type FrontReqPayload struct {
	Data string
}

type CodingReqPayload struct {
	Data        []byte
	Segment_num int32
	Segment_cnt int32
}

// requests

type FrontReq struct {
	Username string
	Time     int64
	Payload  FrontReqPayload
}

type CodingReq struct {
	Username string
	Time     int64
	Payload  CodingReqPayload
}
