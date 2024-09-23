package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"io"
)

func generateID() string {
	b := make([]byte, 32)
	io.ReadFull(rand.Reader, b)
	return hex.EncodeToString(b)
}

func hashKey(key string) string {
	hash := md5.Sum([]byte(key))

	return hex.EncodeToString(hash[:])
}

func newEncryptKey() []byte {
	keyBuf := make([]byte, 32)
	io.ReadFull(rand.Reader, keyBuf)
	return keyBuf
}

func copyDecrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	//Read the IV from the given io.Reader which,
	// in our case should be the block.BlockSize() bytes we read
	iv := make([]byte, block.BlockSize())
	if _, err := src.Read(iv); err != nil {
		return 0, err
	}
	stream := cipher.NewCTR(block, iv)
	return copyStream(stream, block.BlockSize(), src, dst)
	//var (
	//	buf    = make([]byte, 32*1024)
	//	stream = cipher.NewCTR(block, iv)
	//	nw     = block.BlockSize()
	//)
	//for {
	//	n, err := src.Read(buf)
	//	if n > 0 {
	//		stream.XORKeyStream(buf, buf[:n])
	//		nn, err := dst.Write(buf[:n])
	//		if err != nil {
	//			return 0, err
	//		}
	//		nw += nn
	//	}
	//	if err == io.EOF {
	//		break
	//	}
	//	if err != nil {
	//		return 0, err
	//	}
	//}
	//return nw, nil
}

func copyStream(stream cipher.Stream, blockSize int, src io.Reader, dst io.Writer) (int, error) {
	var (
		buf = make([]byte, 32*1024)
		nw  = blockSize
	)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			stream.XORKeyStream(buf, buf[:n])
			nn, err := dst.Write(buf[:n])
			if err != nil {
				return 0, err
			}
			nw += nn
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}
	return nw, nil
}

func copyEncrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	iv := make([]byte, block.BlockSize())

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return 0, err
	}
	// 32*1024 is the max size of copyBuffer()

	// prepend the IV to the file
	if _, err := dst.Write(iv); err != nil {
		return 0, err
	}

	stream := cipher.NewCTR(block, iv)
	return copyStream(stream, block.BlockSize(), src, dst)
	//for {
	//	n, err := src.Read(buf)
	//	if n > 0 {
	//		stream.XORKeyStream(buf, buf[:n])
	//		nn, err := dst.Write(buf[:n])
	//		if err != nil {
	//			return 0, err
	//		}
	//		nw += nn
	//	}
	//
	//	if err == io.EOF {
	//		break
	//	}
	//	if err != nil {
	//		return 0, err
	//	}
	//}

	//return nw, nil
}
