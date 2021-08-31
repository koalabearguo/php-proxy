package main

import (
	"bufio"
	//"bytes"
	"log"
	"net/http"
	"strconv"
	//"strings"
	//"io"
	"time"
)

type response struct {
	res *http.Response
	cfg *config
}

var ResDeleteHeader = []string{"Upgrade", "Alt-Svc", "Alternate-Protocol", "Expect-CT"}

func (res *response) parse_response() *http.Response {
	//
	if res.cfg.Debug {
		for k, v := range res.res.Header {
			for _, value := range v {
				log.Print(k + ":" + value)
			}
		}
		for _, chunkstr := range res.res.TransferEncoding {
			log.Print("Transfer-Encoding" + ":" + chunkstr)
		}
	}
	encrypt := &encrypt{cfg: res.cfg}
	//
	if res.res.StatusCode != http.StatusOK {
		return res.res
	}
	start := time.Now()
	if res.res.Header.Get("Content-Type") == "image/gif" && res.res.Body != nil {
		res.res.Body = encrypt.decrypt_reader(res.res.Body)
	}
	if res.cfg.Debug {
		elapsed := time.Since(start)
		log.Println("First Data Decrypt Time:", elapsed)
	}
	//
	//return res.res //for test
	resp, err := http.ReadResponse(bufio.NewReader(res.res.Body), res.res.Request)
	if err != nil {
		log.Println(err)
		return res.res
	}
	//
	if resp.Header.Get("Content-Length") == "" && resp.ContentLength >= 0 {
		resp.Header.Set("Content-Length", strconv.FormatInt(resp.ContentLength, 10))
	}
	//
	for _, h := range ResDeleteHeader {
		resp.Header.Del(h)
	}
	//
	return resp
}
