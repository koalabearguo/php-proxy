package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
)

type proxy struct {
	//global config
	cfg *config
	//proxy server listener
	listenter *net.Listener
	//php client
	client *client
	//ca sign ssl cert for middle intercept
	signer *CaSigner
	//ca root cert info for middle attack check
	cert *x509.Certificate
}

func (prx *proxy) load_ca() []byte {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	raw, err1 := ioutil.ReadFile(dir + "/php-proxy.crt")
	if err1 != nil {
		return nil
	}
	//log.Print("Load ca cert from ./php-proxy.crt file")
	return raw
}

func (prx *proxy) load_key() []byte {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	raw, err1 := ioutil.ReadFile(dir + "/php-proxy.key")
	if err1 != nil {
		return nil
	}
	//log.Print("Load ca key from ./php-proxy.key file")
	return raw
}

func (prx *proxy) init_ca() {
	//
	var use_ca, use_key []byte
	prx.signer = NewCaSignerCache(1024)
	cert := prx.load_ca()
	key := prx.load_key()
	if cert != nil && key != nil {
		use_ca = cert
		use_key = key
		log.Print("Using external customize CA file:./php-proxy.crt ./php-proxy.key")

	} else {
		use_ca = CaCert
		use_key = CaKey
		log.Print("Using internal Php-Proxy CA file")
	}
	ca, err := tls.X509KeyPair(use_ca, use_key)
	if err != nil {
		log.Fatal(err)
	} else {
		prx.signer.Ca = &ca
	}
	//parse our own php-proxy ca to get info
	prx.cert, err = x509.ParseCertificate(ca.Certificate[0])
	if err != nil {
		log.Fatal(err)
	}
}

func (prx *proxy) init_proxy() {
	//
	ln, err := net.Listen("tcp", prx.cfg.Listen)
	prx.listenter = &ln
	if err != nil {
		log.Panic(err)
	}
	//
	prx.init_ca()
	//

	log.Println("HTTP Proxy Listening on " + prx.cfg.Listen)

	//connect php server config
	prx.client = &client{cfg: prx.cfg}
	prx.client.cert = prx.cert
	prx.client.init_client()

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
	cer := prx.signer.SignHost(host)
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
	Res, err = prx.client.Post(prx.cfg.Fetchserver, "application/octet-stream", req_op.body_buf)
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
