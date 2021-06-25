package main

import (
	"bufio"
	"bytes"
	"io"
	"log"
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
	if res.res.Header.Get("Content-Type") == "image/gif" && res.res.StatusCode == http.StatusOK {
		encrypt.content_decrypt(res.body_buf, res.res.Body)
	} else {
		io.Copy(res.body_buf, res.res.Body)
	}
	//for debug
	if res.cfg.debug == true {
		res_buf := bytes.NewReader(res.body_buf.Bytes())
		res_buf_rd := bufio.NewReader(res_buf)
		Res, err := http.ReadResponse(res_buf_rd, nil)
		if err != nil {
			log.Println(err)
		}
		for k, v := range Res.Header {
			log.Printf(k + ": " + v[0])
		}
	}
	res.res.Body.Close()
}
