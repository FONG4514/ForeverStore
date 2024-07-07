package main

import (
	"bytes"
	"testing"
)

func TestNewStore(t *testing.T) {
	opts := StoreOpts{PathTransformFunc: DefaultPathTransform}
	s := NewStore(opts)
	data := bytes.NewReader([]byte("ssssooooommmmmeeeeDDDDDaaaaattttttaaaaa"))
	if err := s.writeStream("Specialdata", data); err != nil {
		t.Error(err)
	}
}

func TestPathTransformFunc(t *testing.T) {
	_
}
