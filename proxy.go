package main

import (
	"bufio"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"time"
)

type proxy struct {
	//global config
	cfg *config
	//proxy server listener
	listenter *net.Listener
	//http.Transport when connect server
	tr *http.Transport
	//tls.Config when connect https server
	tlsconfig *tls.Config
	//http.Client used to connect server
	client *http.Client
}

func (prx *proxy) init_proxy() {
	//
	ln, err := net.Listen("tcp", prx.cfg.listen)
	prx.listenter = &ln
	if err != nil {
		log.Panic(err)
	}

	log.Println("HTTP Proxy Listening on " + prx.cfg.listen)

	//connect php server config
	prx.init_cfg()

	for {
		client, err := ln.Accept()
		if err != nil {
			log.Panic(err)
		}
		go prx.handleClientRequest(client)
	}
}

func (prx *proxy) handleClientConnectRequest(client net.Conn, host string) (tlscon *tls.Conn, err error) {
	//
	cer := prx.cfg.signer.SignHost(host)
	//
	config := &tls.Config{
		Certificates: []tls.Certificate{*cer},
		MinVersion:   tls.VersionTLS12,
	}
	tlscon = tls.Server(client, config)
	err = tlscon.Handshake()
	if err != nil {
		//log.Println(err)
		return tlscon, err
	}
	return tlscon, nil
}

func (prx *proxy) handleClientRequest(client net.Conn) {
	if client == nil {
		return
	}
	req_op := &request{cfg: prx.cfg}
	//
	Req, err := http.ReadRequest(bufio.NewReader(client))
	http_req := Req
	if err != nil {
		log.Println(err)
		client.Close()
		return
	}
	//
	req_op.http_req = http_req
	//STRIP connect method
	var tlscon *tls.Conn
	if Req.Method == http.MethodConnect {
		_, err = client.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
		if err != nil {
			log.Println(err)
			client.Close()
			return
		}
		hostname := Req.URL.Hostname()
		tlscon, err = prx.handleClientConnectRequest(client, hostname)
		if err != nil {
			log.Println(err)
			tlscon.Close()
			return
		}
		Req, err = http.ReadRequest(bufio.NewReader(tlscon))
		if err != nil {
			log.Println(err)
			tlscon.Close()
			return
		}
		req_op.https_req = Req
	} else if !Req.URL.IsAbs() {
		_, err = client.Write([]byte("HTTP/1.0 200 OK\r\n\r\nThis is php-proxy client."))
		client.Close()
		return
	}
	//
	defer func() {
		if http_req.Method == http.MethodConnect {
			tlscon.Close()
		} else {
			client.Close()
		}
	}()
	//parse http request
	req_op.parse_request()
	//
	//connect php server
	var Res *http.Response
	Res, err = prx.client.Post(prx.cfg.fetchserver, "application/octet-stream", req_op.body_buf)
	if err != nil {
		log.Println(err)
		return
	}
	//
	proxy_res_data := &response{res: Res, cfg: prx.cfg}
	proxy_res_data.parse_response()
	//log.Printf("%q",proxy_res_data.body_buf)
	if http_req.Method == http.MethodConnect {
		_, err = tlscon.Write(proxy_res_data.body_buf.Bytes())
	} else {
		_, err = client.Write(proxy_res_data.body_buf.Bytes())
	}
	if err != nil {
		log.Println(err)
		return
	}
	//
	if http_req.Method == http.MethodConnect {
		tlscon.Close()
	} else {
		client.Close()
	}
	//
}
func (prx *proxy) init_cfg() {
	//tls config
	prx.tlsconfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if prx.cfg.sni != "" {
		prx.tlsconfig.ServerName = prx.cfg.sni
	}
	//tr http.client default tr + tlsconfig
	prx.tr = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       prx.tlsconfig,
	}
	//
	prx.client = &http.Client{
		Transport: prx.tr,
	}
}
