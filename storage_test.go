package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"testing"
)

func TestNewStore(t *testing.T) {
	s := newStore()
	defer teardown(t, s)
	count := 10
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("test_%d", i)
		data := []byte("some bytes")
		log.Printf("this is %d times", i+1)
		if _, err := s.writeStream(key, bytes.NewReader(data)); err != nil {
			t.Error(err)
		}
		if ok := s.Has(key); !ok {
			t.Errorf("damn!!!")
		}
		_, r, err := s.Read(key)
		if err != nil {
			t.Error(err)
		}
		b, _ := ioutil.ReadAll(r)
		if string(b) != string(data) {
			t.Errorf("want %s have %s", data, b)
		}
		if err := s.Delete(key); err != nil {
			t.Errorf("excepted to not have key: %v", key)
		}
	}
}

func TestPathTransformFunc(t *testing.T) {
	key := "yanjingliang"
	pathname := CASPathTransformFunc(key)
	expectedName := "7c154/086cf/3a784/002a7/e0ac0/462dc/71ef8/ab930"
	expectedFileName := "7c154086cf3a784002a7e0ac0462dc71ef8ab930"
	if pathname.Pathname != expectedName {
		t.Errorf("Error %s", pathname)
	}
	if pathname.Filename != expectedFileName {
		t.Errorf("Error happen %s", pathname)
	}
}

func TestDelete(t *testing.T) {
	opts := StoreOpts{PathTransformFunc: CASPathTransformFunc}
	s := NewStore(opts)
	key := "FileData"
	data := []byte("some bytes")

	if _, err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}
	if err := s.Delete(key); err != nil {
		t.Error(err)
	}
}

func newStore() *Store {
	opts := StoreOpts{PathTransformFunc: CASPathTransformFunc}
	s := NewStore(opts)
	return s
}

func teardown(t *testing.T, s *Store) {
	if err := s.Clear(); err != nil {
		t.Error(err)
	}
}
