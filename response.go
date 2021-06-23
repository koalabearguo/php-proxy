package main

import (
	"bytes"
	"io"
	"net/http"
)

type response struct {
	res      *http.Response
	cfg      *config
	body_buf *bytes.Buffer
}

func (res *response) parse_response() {
	res.body_buf = bytes.NewBuffer(nil)
	//
	encrypt := &encrypt{cfg: res.cfg}
	//
	if len(res.res.Header["Content-Type"]) != 0 && res.res.Header["Content-Type"][0] == "image/gif" && res.res.StatusCode == http.StatusOK {
		encrypt.content_decrypt(res.body_buf, res.res.Body)
	} else {
		io.Copy(res.body_buf, res.res.Body)
	}
	res.res.Body.Close()
}
