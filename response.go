package main

import (
	"bufio"
	"bytes"
	"log"
	"net/http"
	"strconv"
	"strings"
	//"io"
	"time"
)

type response struct {
	res      *http.Response
	cfg      *config
	body_buf *bytes.Buffer
}

func (res *response) parse_response() *http.Response {
	body_buf_tmp := bytes.NewBuffer(nil)
	res.body_buf = bytes.NewBuffer(nil)
	//
	encrypt := &encrypt{cfg: res.cfg}
	//
	start := time.Now()
	if res.res.Header.Get("Content-Type") == "image/gif" && res.res.StatusCode == http.StatusOK {
		//encrypt.content_decrypt(body_buf_tmp, res.res.Body)
		res.res.Body = encrypt.decrypt_reader(res.res.Body)
	} else {
		//io.Copy(body_buf_tmp, res.res.Body)
		res.res.Body = res.res.Body
	}
	elapsed := time.Since(start)
	log.Println("Copy Data Time elapsed:", elapsed)
	//res.res.Body.Close()
	//
	//io.Copy(body_buf_tmp, res.res.Body)
	only_body := body_buf_tmp.Bytes()
	//log.Printf("%q", only_body)
	//return nil
	res_buf_rd := bufio.NewReader(res.res.Body)
	resp, err := http.ReadResponse(res_buf_rd, nil)
	if err != nil {
		log.Println(err)
		res.body_buf.Write(only_body)
		return nil
	}
	if resp.Header.Get("Content-Length") == "" && resp.ContentLength >= 0 {
		resp.Header.Set("Content-Length", strconv.FormatInt(resp.ContentLength, 10))
	}
	resp.Header.Del("Upgrade")
	resp.Header.Del("Alt-Svc")
	resp.Header.Del("Alternate-Protocol")
	resp.Header.Del("Expect-CT")
	return resp
	Res := resp
	//process header
	Res.Header.Del("Upgrade")
	Res.Header.Del("Alt-Svc")
	Res.Header.Del("Alternate-Protocol")
	Res.Header.Del("Expect-CT")
	//
	err = nil
	pure_body := bytes.NewBuffer(nil)
	n, err := pure_body.ReadFrom(Res.Body)
	n = n
	chunked_str := Res.Header.Get("Transfer-Encoding")
	if strings.Contains(chunked_str, "chunked") == false {
		if Res.Header.Get("Content-Length") == "" && Res.ContentLength != -1 {
			Res.Header.Add("Content-Length", strconv.FormatInt(Res.ContentLength, 10))
		} /*else {
			Res.Header.Set("Content-Length", strconv.FormatInt(n, 10))
		}*/
	}
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
	//res.body_buf.ReadFrom(Res.Body)
	res.body_buf.Write(pure_body.Bytes())
	Res.Body.Close()
	//
	return nil
}
