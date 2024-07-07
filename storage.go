package main

import (
	"crypto/sha1"
	"io"
	"log"
	"os"
)

type PathTransformFunc func(string) string

type StoreOpts struct {
	PathTransformFunc PathTransformFunc
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	return &Store{
		StoreOpts: opts,
	}

}

func (s *Store) writeStream(key string, r io.Reader) error {
	pathName := s.PathTransformFunc(key)
	if err := os.MkdirAll(pathName, os.ModePerm); err != nil {
		return err
	}

	fileName := "testfile"
	FullFilePath := pathName + "/" + fileName

	f, err := os.Create(FullFilePath)
	if err != nil {
		return err
	}

	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}

	log.Printf("writen (%d) bytes to disk：%s", n, FullFilePath)
	return nil
}

func DefaultPathTransform(key string) string {
	return key
}

func CASPathTransformFunc(key string) string {
	hash := sha1.Sum([]byte(key))
}
