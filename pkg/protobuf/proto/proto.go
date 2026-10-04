package proto

import real "google.golang.org/protobuf/proto"

type Message = real.Message

var (
	Marshal   = real.Marshal
	Unmarshal = real.Unmarshal
	Clone     = real.Clone
	Equal     = real.Equal
	Reset     = real.Reset
)
