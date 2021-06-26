package main

import (
	"bytes"
	"compress/flate"
	"io"
	"log"
	"net/http"
	"strconv"
)

type request struct {
	//http request header
	http_req *http.Request
	//https request header
	https_req *http.Request
	//POST body buf
	body_buf *bytes.Buffer
	//
	cfg *config
}

func (req *request) parse_request() {
	var real_req *http.Request
	if req.http_req.Method == http.MethodConnect {
		real_req = req.https_req
	} else {
		real_req = req.http_req
	}
	//
	com := &compress{cfg: req.cfg}
	//com.level = flate.NoCompression
	com.level = flate.BestSpeed
	//
	//process body
	deflare_body_buf := bytes.NewBuffer(nil)
	if real_req.ContentLength > 0 && real_req.Header["Content-Encoding"] == nil {
		com.deflate_compress(deflare_body_buf, real_req.Body)
		real_req.Header.Del("Content-Length")
		real_req.Header.Add("Content-Encoding", "deflate")
		real_req.Header.Add("Content-Length", strconv.Itoa(deflare_body_buf.Len()))
	} else {
		io.Copy(deflare_body_buf, real_req.Body)
	}
	if real_req.Header.Get("Host") == "" {
		if req.http_req.Method == http.MethodConnect {
			real_req.Header.Add("Host", req.http_req.URL.Host)
		} else {
			real_req.Header.Add("Host", real_req.URL.Host)
		}
	}
	real_req.Body.Close()
	//process header
	header_buf := bytes.NewBuffer(nil)
	deflare_header_buf := bytes.NewBuffer(nil)
	var req_line string
	if req.http_req.Method == http.MethodConnect {
		req_line = real_req.Method + " " + "https:" + req.http_req.URL.String() + real_req.URL.String() + " " + real_req.Proto
	} else {
		req_line = real_req.Method + " " + real_req.URL.String() + " " + real_req.Proto
	}
	log.Print("PHP " + req_line)
	_, err := header_buf.WriteString(req_line + "\r\n")
	if err != nil {
		log.Printf("%s", err)
	}
	//
	real_req.Header.Add("X-URLFETCH-password", req.cfg.password)
	//
	real_req.Header.Del("Proxy-Authorization")
	real_req.Header.Del("Proxy-Connection")
	//
	//disable websocket upgrade
	real_req.Header.Del("Upgrade")
	real_req.Header.Del("Sec-Websocket-Key")
	real_req.Header.Del("Sec-Websocket-Version")
	real_req.Header.Del("Sec-Websocket-Extensions")
	real_req.Header.Del("Sec-WebSocket-Protocol")
	if real_req.Header.Get("Connection") == "Upgrade" {
		real_req.Header.Del("Connection")
	}

	for k, v := range real_req.Header {
		_, err = header_buf.WriteString(k + ": " + v[0] + "\r\n")
		if req.cfg.debug == true {
			log.Print(k + ": " + v[0])
		}
		if err != nil {
			log.Printf("%s", err)
		}
	}
	com.deflate_compress(deflare_header_buf, header_buf)
	//pack (header length may biger than 65536 bytes)
	var length [2]byte
	if deflare_header_buf.Len() < 65536 {
		length[0] = byte(deflare_header_buf.Len() / 256)
		length[1] = byte(deflare_header_buf.Len() % 256)
	} else {
		log.Fatal("request header too big")
	}
	//
	req.body_buf = bytes.NewBuffer(length[:2])
	req.body_buf.Write(deflare_header_buf.Bytes())
	_, err = req.body_buf.Write(deflare_body_buf.Bytes())
	if err != nil {
		log.Printf("%s", err)
	}

}
