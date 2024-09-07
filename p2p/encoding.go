package p2p

import (
	"encoding/gob"
	"io"
)

type Decoder interface {
	Decode(io.Reader, *RPC) error
}

type GOBDecoder struct {
}

func (dec GOBDecoder) Decode(r io.Reader, message *RPC) error {
	return gob.NewDecoder(r).Decode(message)
}

type DefaultDecoder struct {
}

func (dec DefaultDecoder) Decode(r io.Reader, message *RPC) error {
	peekBuf := make([]byte, 1)
	if _, err := r.Read(peekBuf); err != nil {
		return nil
	}
	stream := peekBuf[0] == IncomingStream
	// In case of stream we are not decoding what is being sent over the network
	// just setting Stream true, so we can handle that in our logic
	if stream {
		message.Stream = true
		return nil
	}

	buf := make([]byte, 2048)
	n, err := r.Read(buf)
	if err != nil {
		return err
	}
	message.Payload = buf[0:n]
	return nil
}
