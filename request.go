package main

import (
	"bytes"
	"compress/flate"
	"io"
	"log"
	"net/http"
	//"strconv"
)

type request struct {
	//http request header
	http_req *http.Request
	//Client request
	cli_req *http.Request
	//cfg
	cfg *config
	//client.Body io.ReadCloser interface impl
	readers     []io.Reader
	multiReader io.Reader
}

var (
	ReqDeleteHeader = map[string]bool{
		//disable websocket upgrade
		"Upgrade":                  true,
		"Sec-Websocket-Key":        true,
		"Sec-Websocket-Version":    true,
		"Sec-Websocket-Extensions": true,
		"Sec-WebSocket-Protocol":   true,
		"Vary":                     true,
		"Via":                      true,
		"X-Forwarded-For":          true,
		"Proxy-Authorization":      true,
		"Proxy-Connection":         true,
		"X-Chrome-Variations":      true,
		"Connection":               true,
		"Cache-Control":            true,
	}
)

func (req *request) MultiReader(readers ...io.Reader) io.ReadCloser {
	req.readers = readers
	req.multiReader = io.MultiReader(readers...)
	return req
}

func (req *request) Read(p []byte) (n int, err error) {
	return req.multiReader.Read(p)
}

func (req *request) Close() (err error) {
	for _, r := range req.readers {
		if c, ok := r.(io.Closer); ok {
			if e := c.Close(); e != nil {
				err = e
			}
		}
	}
	return err
}

func (req *request) parse_request() {
	//
	com := &compress{cfg: req.cfg}
	//com.level = flate.NoCompression
	//com.level = flate.BestSpeed
	com.level = flate.BestCompression
	//
	//process header
	header_buf := bytes.NewBuffer(nil)
	deflare_header_buf := bytes.NewBuffer(nil)
	var req_line string
	req_line = req.http_req.Method + " " + req.http_req.URL.String() + " " + req.http_req.Proto
	log.Print("PHP " + req_line)
	_, err := header_buf.WriteString(req_line + "\r\n")
	if err != nil {
		log.Printf("%s", err)
	}
	//
	req.http_req.Header.Add("X-URLFETCH-password", req.cfg.Password)
	//
	//for feature use(index.php need upgrade)
	if req.cfg.Insecure {
		req.http_req.Header.Add("X-URLFETCH-insecure", "1")
	}
	//
	req.http_req.Header.WriteSubset(header_buf, ReqDeleteHeader)
	//
	if req.cfg.Debug {
		for k, v := range req.http_req.Header {
			for _, value := range v {
				log.Print(k + ":" + value)
			}
		}
	}
	//
	com.deflate_compress(deflare_header_buf, header_buf)
	//
	//pack (header length may biger than 65536 bytes)
	var length [2]byte
	if deflare_header_buf.Len() < 65536 {
		length[0] = byte(deflare_header_buf.Len() / 256)
		length[1] = byte(deflare_header_buf.Len() % 256)
	} else {
		log.Fatal("request header too big")
	}
	//
	req.cli_req = &http.Request{
		Method: http.MethodPost,
		Header: http.Header{},
	}
	//default use brower UA
	req.cli_req.Header.Set("User-Agent", req.http_req.Header.Get("User-Agent"))
	//
	if req.http_req.ContentLength > 0 {
		req.cli_req.ContentLength = int64(len(length)+deflare_header_buf.Len()) + req.http_req.ContentLength
		req.cli_req.Body = req.MultiReader(bytes.NewReader(length[:]), deflare_header_buf, req.http_req.Body)
	} else {
		req.cli_req.ContentLength = int64(len(length) + deflare_header_buf.Len())
		req.cli_req.Body = req.MultiReader(bytes.NewReader(length[:]), deflare_header_buf)
	}
	//

}
