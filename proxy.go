package main

import (
	"bufio"
	//"context"
	"crypto/tls"
	"crypto/x509"
	//"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type proxy struct {
	//global config
	cfg *config
	//prepare static buf
	bufpool sync.Pool
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
	ca_path := ""
	if prx.cfg.Ca == "" {
		ca_path = dir + "/php-proxy.crt"
	} else {
		ca_path = prx.cfg.Ca
	}
	raw, err1 := ioutil.ReadFile(ca_path)
	if err1 != nil {
		return nil
	}
	log.Print("Load ca cert from " + ca_path + " file")
	return raw
}

func (prx *proxy) load_key() []byte {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	key_path := ""
	if prx.cfg.Key == "" {
		key_path = dir + "/php-proxy.key"
	} else {
		key_path = prx.cfg.Key
	}
	raw, err1 := ioutil.ReadFile(key_path)
	if err1 != nil {
		return nil
	}
	log.Print("Load ca key from " + key_path + " file")
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
		log.Print("Using external customize CA file")

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
	//prepare gen google cert for cache(not must)
	_ = prx.signer.SignHost("www.google.com")
	_ = prx.signer.SignHost("www.youtube.com")
	_ = prx.signer.SignHost("www.googlevideo.com")
	_ = prx.signer.SignHost("www.gstatic.com")
	_ = prx.signer.SignHost("www.ggpht.com")
}

func (prx *proxy) init_proxy() {
	//
	prx.init_ca()
	//

	log.Println("HTTP Proxy Listening on " + prx.cfg.Listen)

	//connect php server config
	prx.client = &client{cfg: prx.cfg}
	prx.client.cert = prx.cert
	prx.client.init_client()
	//
	prx.bufpool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 32*1024)
		},
	}
	//
	log.Fatal(http.ListenAndServe(prx.cfg.Listen, prx))
	//

}

func (prx *proxy) IOCopy(dst io.Writer, src io.Reader) (written int64, err error) {
	//not use tmp mem,use prepared mem
	buf := prx.bufpool.Get().([]byte)
	written, err = io.CopyBuffer(dst, src, buf)
	prx.bufpool.Put(buf)
	return written, err
}
func (prx *proxy) ServePROXY(rw http.ResponseWriter, req *http.Request) {
	hijacker, ok := rw.(http.Hijacker)
	if !ok {
		log.Println("Not Support Hijacking")
	}
	client, _, err := hijacker.Hijack()
	if err != nil {
		log.Println(err)
	}
	defer client.Close()
	//
	var address string
	if strings.Index(req.Host, ":") == -1 { //host port not include,default 80
		address = req.Host + ":http"
	} else {
		address = req.Host
	}

	server, err := net.Dial("tcp", address)
	if err != nil {
		log.Println(err)
		return
	}
	defer server.Close()
	//
	if req.Method == http.MethodConnect {
		io.WriteString(client, "HTTP/1.1 200 Connection established\r\n\r\n")
		//exchange data
		go prx.IOCopy(server, client)
		prx.IOCopy(client, server)
		return
	}
	//
	req.Header.Del("Proxy-Authorization")
	req.Header.Del("Proxy-Connection")
	//
	Req := req
	//http proxy keep alive
	for true {
		err = Req.Write(server)
		if err != nil {
			return
		}
		Res, err := http.ReadResponse(bufio.NewReader(server), Req)
		if err != nil {
			return
		}
		err = Res.Write(client)
		if err != nil {
			return
		}
		Req, err = http.ReadRequest(bufio.NewReader(client))
		if err != nil {
			return
		}
		//
		req.Header.Del("Proxy-Authorization")
		req.Header.Del("Proxy-Connection")
		//
	}

}
func (prx *proxy) isblocked(host string) bool {
	hostname := stripPort(host)
	hostnamelth := len(hostname)
	for key, _ := range gfwlist {
		if hostnamelth >= len(key) {
			subhost := hostname[(hostnamelth - len(key)):]
			if key == subhost {
				return true
			}
		}
	}
	return false

}
func (prx *proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	var tlscon *tls.Conn
	//
	if prx.cfg.Autoproxy && (req.Method == http.MethodConnect || req.Method != http.MethodConnect && req.URL.IsAbs()) {
		blocked := prx.isblocked(req.Host)
		if blocked == false {
			log.Printf("Direct Connect %s", req.Host)
			prx.ServePROXY(rw, req)
			return
		}
	}
	//
	if req.Method != http.MethodConnect && !req.URL.IsAbs() {
		//
		req.URL.Scheme = "https"
		if req.Host == "" {
			req.URL.Host = "localhost"
		} else {
			req.URL.Host = req.Host
		}
		if prx.cfg.Debug {
			log.Printf("Request Host:%s", req.URL.Host)
		}

	}
	//
	//Strip ssl
	if req.Method == http.MethodConnect {
		hijacker, ok := rw.(http.Hijacker)
		if !ok {
			if req.Body != nil {
				req.Body.Close()
			}
			log.Println("Not Support Hijacking")
			return
		}
		conn, _, err := hijacker.Hijack()
		if err != nil {
			if req.Body != nil {
				req.Body.Close()
			}
			log.Println(err)
			return
		}

		_, err = io.WriteString(conn, "HTTP/1.1 200 Connection established\r\n\r\n")
		if err != nil {
			log.Println(err)
			conn.Close()
			return
		}

		tlscon, err = prx.handleClientConnectRequest(conn, req.URL.Hostname())
		if err != nil {
			log.Println(err)
			tlscon.Close()
			return
		}
		loConn, err := net.Dial("tcp", prx.cfg.Listen)
		if err != nil {
			log.Println(err)
			tlscon.Close()
			return
		}

		go prx.IOCopy(tlscon, loConn)
		prx.IOCopy(loConn, tlscon)
		err = tlscon.Close()
		if err != nil {
			log.Println(err)
		}
		err = loConn.Close()
		if err != nil {
			log.Println(err)
		}
		return

	}
	//
	req_op := &request{cfg: prx.cfg, http_req: req}
	//
	//parse http request
	start := time.Now()
	req_op.parse_request()
	if prx.cfg.Debug == true {
		elapsed := time.Since(start)
		log.Println("HTTP POST body Proc Time:", elapsed)
	}
	//
	//connect php server
	start = time.Now()
	Res, err := prx.client.Do(req_op.cli_req)
	if prx.cfg.Debug == true {
		elapsed := time.Since(start)
		log.Println("HTTP POST Time:", elapsed)
	}
	if err != nil {
		log.Println(err)
		if prx.client.tr3 != nil {
			prx.client.tr3.Close()
		}
		origin := req_op.http_req.Header.Get("Origin")
		if origin != "" {
			rw.Header().Add("Access-Control-Allow-Origin", origin)
			rw.Header().Add("Access-Control-Allow-Credentials", "true")
		}
		http.Error(rw, "empty response", http.StatusBadGateway)
		return
	}
	//
	defer Res.Body.Close()
	//
	proxy_res_data := &response{res: Res, cfg: prx.cfg}
	resp := proxy_res_data.parse_response()

	if resp == nil {
		log.Println("Response is nil")
		return
	}

	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			rw.Header().Add(key, value)
			if prx.cfg.Debug {
				log.Print(key + ":" + value)
			}
		}
	}
	//Patch CORS
	origin := req_op.http_req.Header.Get("Origin")
	if origin != "" && rw.Header().Get("Access-Control-Allow-Origin") == "" {
		rw.Header().Add("Access-Control-Allow-Origin", origin)
	}
	if origin != "" && rw.Header().Get("Access-Control-Allow-Credentials") == "" {
		rw.Header().Add("Access-Control-Allow-Credentials", "true")
	}
	//rw.Header().Set("Set-Cookie", rw.Header().Get("Set-Cookie") + ";HttpOnly;Secure;SameSite=Strict" )
	//
	rw.WriteHeader(resp.StatusCode)
	_, err = prx.IOCopy(rw, resp.Body)
	//
	if err != nil {
		if strings.Contains(err.Error(), io.ErrUnexpectedEOF.Error()) == true {
			hijacker, ok := rw.(http.Hijacker)
			if !ok {
				log.Println("Not Support Hijacking")
			}
			conn, _, err := hijacker.Hijack()
			if err != nil {
				log.Println(err)
			}
			err = conn.Close()
			if err != nil {
				log.Println(err)
			}

		} else if strings.Contains(err.Error(), "invalid byte in chunk length") == true {
			hijacker, ok := rw.(http.Hijacker)
			if !ok {
				log.Println("Not Support Hijacking")
			}
			conn, _, err := hijacker.Hijack()
			if err != nil {
				log.Println(err)
			}
			err = conn.Close()
			if err != nil {
				log.Println(err)
			}
		} else {
			log.Println(err)
		}
	}
}
func (prx *proxy) handleClientConnectRequest(client net.Conn, host string) (tlscon *tls.Conn, err error) {
	//
	cer := prx.signer.SignHost(host)
	//
	config := &tls.Config{
		Certificates: []tls.Certificate{*cer},
	}
	tlscon = tls.Server(client, config)
	err = tlscon.Handshake()
	if err != nil {
		return tlscon, err
	}
	return tlscon, nil
}
