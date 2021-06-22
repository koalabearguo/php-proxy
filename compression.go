package main

import (
	"bytes"
	"compress/flate"
	"io"
	"log"
)

type compress struct {
	cfg   *config
	level int
}

func (c *compress) deflate_compress(dst_buf *bytes.Buffer, src_buf io.Reader) {

	flateWrite, _ := flate.NewWriter(dst_buf, c.level)
	if _, err := io.Copy(flateWrite, src_buf); err != nil {
		if err != io.ErrUnexpectedEOF {
			log.Print(err)
		}
	}
	//
	flateWrite.Flush()
	flateWrite.Close()
	//
}
