package main

import (
	"bytes"
	"io"
	"io/ioutil"
)

type encrypt struct {
	cfg *config
	rc  io.ReadCloser
}

func (e *encrypt) content_decrypt(dst_buf *bytes.Buffer, src_buf io.Reader) {

	var key []byte
	b, _ := ioutil.ReadAll(src_buf)
	if e.cfg.Password != "" {
		key = []byte(e.cfg.Password)
	} else {
		key = []byte{0}
	}
	for i := 0; i < len(b); i++ {
		b[i] = b[i] ^ key[0]
	}
	dst_buf.Write(b)

}
func (e *encrypt) decrypt_reader(src_buf io.ReadCloser) io.ReadCloser {
	e.rc = src_buf
	return e
}

func (e *encrypt) Close() error {
	return e.rc.Close()
}

func (e *encrypt) Read(p []byte) (n int, err error) {
	var key []byte
	n, err = e.rc.Read(p)
	if e.cfg.Password != "" {
		key = []byte(e.cfg.Password)
	} else {
		key = []byte{0}
	}
	for i := 0; i < n; i++ {
		p[i] ^= key[0]
	}

	return n, err
}
