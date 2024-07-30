package main

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func TestNewStore(t *testing.T) {
	opts := StoreOpts{PathTransformFunc: CASPathTransformFunc}
	s := NewStore(opts)
	key := "FileData"
	data := []byte("some bytes")
	if err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}
	if ok := s.Has(key); !ok {
		t.Errorf("damn!!!")
	}
	r, err := s.Read(key)
	if err != nil {
		t.Error(err)
	}
	b, _ := ioutil.ReadAll(r)
	if string(b) != string(data) {
		t.Errorf("want %s have %s", data, b)
	}
	s.Delete(key)
}

func TestPathTransformFunc(t *testing.T) {
	key := "yanjingliang"
	pathname := CASPathTransformFunc(key)
	expectedName := "7c154/086cf/3a784/002a7/e0ac0/462dc/71ef8/ab930"
	if pathname.Pathname != expectedName {
		t.Errorf("Error %s", pathname)
	}
}

func TestDelete(t *testing.T) {
	opts := StoreOpts{PathTransformFunc: CASPathTransformFunc}
	s := NewStore(opts)
	key := "FileData"
	data := []byte("some bytes")

	if err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}
	if err := s.Delete(key); err != nil {
		t.Error(err)
	}
}
