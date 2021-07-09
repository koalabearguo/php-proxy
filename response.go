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
	body_buf_tmp := bytes.NewBuffer(nil)
	res.body_buf = bytes.NewBuffer(nil)
	//
	encrypt := &encrypt{cfg: res.cfg}
	//
	if res.res.Header.Get("Content-Type") == "image/gif" && res.res.StatusCode == http.StatusOK {
		encrypt.content_decrypt(body_buf_tmp, res.res.Body)
	} else {
		io.Copy(body_buf_tmp, res.res.Body)
	}
	res.res.Body.Close()
	//
	only_body := body_buf_tmp.Bytes()
	res_buf_rd := bufio.NewReader(body_buf_tmp)
	Res, err := http.ReadResponse(res_buf_rd, nil)
	if err != nil {
		log.Println(err)
		res.body_buf.Write(only_body)
		return
	}
	//process header
	Res.Header.Del("Upgrade")
	Res.Header.Del("Alt-Svc")
	Res.Header.Del("Alternate-Protocol")
	Res.Header.Del("Expect-CT")
	//
	res.body_buf.WriteString(Res.Proto + " " + Res.Status + "\r\n")
	for k, v := range Res.Header {
		//for debug
		if res.cfg.Debug {
			log.Print(k + ": " + v[0])
		}
		res.body_buf.WriteString(k + ": " + v[0] + "\r\n")
	}
	res.body_buf.WriteString("\r\n")
	res.body_buf.ReadFrom(Res.Body)
	Res.Body.Close()
	//
}
