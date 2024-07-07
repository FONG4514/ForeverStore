package p2p

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTCPTransport(t *testing.T) {
	opts := TCPTransportOpts{
		ListenAddress:  ":3000",
		ShakeHandsFunc: NOPHandshakerFunc,
		Decoder:        DefaultDecoder{},
	}
	tr := NewTCPTransport(opts)
	assert.Equal(t, tr.ListenAddress, ":3000")
	assert.Nil(t, tr.ListenAndAccept())
}
