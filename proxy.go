package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
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
	prx.init_ca()
	//

	log.Println("HTTP Proxy Listening on " + prx.cfg.Listen)

	//connect php server config
	prx.client = &client{cfg: prx.cfg}
	prx.client.cert = prx.cert
	prx.client.init_client()
	//
	log.Fatal(http.ListenAndServe(prx.cfg.Listen, prx))
	//

}

func (prx *proxy) new_proxy() {
	//
}

func test_http_buf(resp *http.Response) {
	reader := bufio.NewReader(resp.Body)

	var buf [8192]byte
	for {
		n, err := reader.Read(buf[:])
		if err != nil {
			log.Printf("%d", n)
			log.Fatal(err)
		} else {
			log.Printf("%d", n)
		}
	}

}

func (prx *proxy) debug_request(r *http.Request) string {
	// Create return string
	var request []string
	// Add the request string
	url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)
	// Add the host
	request = append(request, fmt.Sprintf("Host: %v", r.Host))
	// Loop through headers
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}

	// If this is a POST, add post data
	if r.Method == "POST" {
		r.ParseForm()
		request = append(request, "\n")
		request = append(request, r.Form.Encode())
	}
	// Return the request as a string
	return strings.Join(request, "\n")
}
func (prx *proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	req_op := &request{cfg: prx.cfg, http_req: req}
	var tlscon *tls.Conn
	//Strip ssl
	if req.Method == http.MethodConnect {
		hijacker, ok := rw.(http.Hijacker)
		if !ok {
			log.Println(fmt.Errorf("%#v does not implments Hijacker", rw))
			return
		}
		hijacker = hijacker
		conn, _, err := hijacker.Hijack()
		if err != nil {
			log.Println(fmt.Errorf("http.ResponseWriter Hijack failed: %s", err))
			return
		}

		_, err = io.WriteString(conn, "HTTP/1.1 200 Connection established\r\n\r\n")
		if err != nil {
			conn.Close()
			log.Println(err)
			return
		}

		//handleClientConnectRequest(conn,req.URL.Hostname())
		tlscon, err = prx.handleClientConnectRequest(conn, req.URL.Hostname())
		if err != nil {
			log.Println(err)
			tlscon.Close()
			return
		}
		req_op.https_req, err = http.ReadRequest(bufio.NewReader(tlscon))
		if err != nil {
			log.Println(err)
			tlscon.Close()
			return
		}
	} else if !req.URL.IsAbs() {
		_, _ = io.WriteString(rw, "HTTP/1.0 200 OK\r\n\r\nThis is php-proxy client.")
		return
	}
	//parse http request
	req_op.parse_request()
	//
	//connect php server
	//var Res *http.Response
	start := time.Now()
	//go test_http_buf(Res)
	Res, err := prx.client.Post(prx.cfg.Fetchserver, "application/octet-stream", req_op.body_buf)
	if prx.cfg.Debug == true {
		elapsed := time.Since(start)
		log.Println("HTTP POST Time elapsed:", elapsed)
	}
	if err != nil {
		log.Println(err)
		return
	}
	//
	proxy_res_data := &response{res: Res, cfg: prx.cfg}
	resp := proxy_res_data.parse_response()
	if resp == nil {
		return
	}
	defer resp.Body.Close()
	//log.Printf("%q",proxy_res_data.body_buf)
	var n, lth int
	lth = proxy_res_data.body_buf.Len()
	if req_op.http_req.Method == http.MethodConnect {
		//n, err = tlscon.Write(proxy_res_data.body_buf.Bytes())
		_, err = tlscon.Write([]byte(Res.Proto + " " + Res.Status + "\r\n"))
		for key, values := range resp.Header {
			for _, value := range values {
				_, err = tlscon.Write([]byte(key + ": " + value + "\r\n"))
			}
		}
		_, err = tlscon.Write([]byte("\r\n"))
		_, err = io.Copy(tlscon, resp.Body)
	} else {
		for key, values := range resp.Header {
			for _, value := range values {
				rw.Header().Add(key, value)
			}
		}
		rw.WriteHeader(resp.StatusCode)
		//n, err = io.Copy(rw, proxy_res_data.body_buf.String())
		_, err = io.Copy(rw, resp.Body)
	}
	if err != nil {
		log.Println(err)
		return
	}
	n = lth
	if n != lth {
		log.Printf("Send Data Length mismatch.%d/%d", n, lth)
		return
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
