package main

import (
	"bytes"
	"io"
	//"log"
	"net/http"
)

type response struct {
	res      *http.Response
	cfg      *config
	body_buf *bytes.Buffer
}

func (res *response) parse_response() {
	res.body_buf = bytes.NewBuffer(nil)
	//log.Printf("%s", res.res.Header)
	encrypt := &encrypt{cfg: res.cfg}
	//test_body_buf := bytes.NewBuffer(nil)
	//io.Copy(test_body_buf, res.res.Body)
	//encrypt.content_decrypt(test_body_buf, res.res.Body)
	//log.Printf("%s",test_body_buf)
	if len(res.res.Header["Content-Type"]) != 0 && res.res.Header["Content-Type"][0] == "image/gif" && res.res.StatusCode == http.StatusOK {
		encrypt.content_decrypt(res.body_buf, res.res.Body)
	} else {
		io.Copy(res.body_buf, res.res.Body)
	}
	res.res.Body.Close()
}
