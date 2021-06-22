package main

import (
	"bufio"
	"crypto/tls"
	"log"
	"net"
	"net/http"
)

type proxy struct {
	//global config
	cfg *config
	//proxy server listener
	listenter *net.Listener
}

func (prx *proxy) init_proxy() {
	//
	ln, err := net.Listen("tcp", prx.cfg.listen)
	prx.listenter = &ln
	if err != nil {
		log.Panic(err)
	}

	log.Println("Listening on " + prx.cfg.listen)

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
	if err != nil {
		//log.Println(err)
		return nil, err
	}
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
	req_op := &request{prx: prx}
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
			if tlscon == nil {
				client.Close()
			} else {
				tlscon.Close()
			}
			return
		}
		Req, err = http.ReadRequest(bufio.NewReader(tlscon))
		if err != nil {
			log.Println(err)
			tlscon.Close()
			return
		}
		req_op.https_req = Req
	} else if (!Req.URL.IsAbs()){
		_, err = client.Write([]byte("HTTP/1.1 200 OK\r\n\r\nThis is php-proxy cilent."))
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
	server := &http.Client{}
	var Res *http.Response
	Res, err = server.Post(prx.cfg.fetchserver, "application/octet-stream", req_op.body_buf)
	if err != nil {
		log.Println(err)
		server.CloseIdleConnections()
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
		server.CloseIdleConnections()
		return
	}
	//
	if http_req.Method == http.MethodConnect {
		tlscon.Close()
	} else {
		client.Close()
	}
	//
	server.CloseIdleConnections()
}
