package main

import (
	//"bufio"
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

func (prx *proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	var tlscon *tls.Conn
	//
	if req.Method != "CONNECT" && !req.URL.IsAbs() {
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
		return
	}
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
		}
	}
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
