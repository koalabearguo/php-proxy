package main

import (
	"bytes"
	"io"
	"io/ioutil"
)

type encrypt struct {
	cfg *config
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
