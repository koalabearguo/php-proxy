package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"time"
)

type client struct {
	//global config
	cfg *config
	//http.Transport when connect server
	tr  *http.Transport
	tr3 *http3.RoundTripper
	//http3 quic config
	qconf *quic.Config
	//tls.Config when connect https server
	tlsconfig *tls.Config
	//http.Client used to connect server
	client *http.Client
	//ca root cert info for middle attack check
	cert *x509.Certificate
	//server
	server *url.URL
	//proxy
	urlproxy *url.URL
}

func (cli *client) Post(url, contentType string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	if cli.cfg.Sni != "" {
		if req.URL.Port() == "" {
			req.Host = cli.cfg.Sni
		} else {
			req.Host = cli.cfg.Sni + ":" + req.URL.Port()
		}
	}
	if cli.cfg.Debug == true {
		for k, v := range req.Header {
			log.Print(k + ": " + v[0])
		}
	}
	return cli.client.Do(req)
	//return cli.client.Post(url, contentType, body)
}

func (cli *client) Dummy_Get(url string) (resp *http.Response, err error) {
	return cli.client.Get(url)
}

func (cli *client) Do(req *http.Request) (resp *http.Response, err error) {
	if req == nil {
		log.Printf("POST Request == nil")
	} else {
		req.URL = cli.server
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	if cli.cfg.User_agent != "" {
		req.Header.Set("User-Agent", cli.cfg.User_agent)
	}
	if cli.cfg.Sni != "" {
		if req.URL.Port() == "" {
			req.Host = cli.cfg.Sni
		} else {
			req.Host = cli.cfg.Sni + ":" + req.URL.Port()
		}
	}
	if cli.cfg.Debug == true {
		for k, v := range req.Header {
			for _, value := range v {
				log.Print(k + ": " + value)
			}
		}
	}
	return cli.client.Do(req)
}

func (cli *client) init_client() {
	//
	server, err := url.Parse(cli.cfg.Fetchserver)
	if err != nil {
		log.Fatal(err)
	}
	cli.server = server
	//
	//tls config
	cli.tlsconfig = &tls.Config{
		InsecureSkipVerify: cli.cfg.Insecure,
		VerifyConnection:   cli.VerifyConnection,
	}
	if cli.cfg.Insecure == true {
		cli.tlsconfig.VerifyConnection = nil
	}
	if cli.cfg.Sni != "" {
		cli.tlsconfig.ServerName = cli.cfg.Sni
	}
	//parent proxy
	if cli.cfg.Proxy != "" {
		cli.urlproxy, err = url.Parse(cli.cfg.Proxy)
		if err != nil {
			log.Println(err)
		}
	}
	if cli.cfg.Http3 == false {
		//tr http.client default tr + tlsconfig
		cli.tr = &http.Transport{
			//Dial: cli.Dial,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig:       cli.tlsconfig,
			WriteBufferSize:       4096 * 32,
			ReadBufferSize:        4096 * 32,
		}
		if cli.urlproxy != nil {
			if cli.urlproxy.Scheme == "https" {
				//for connect https proxy server
				cli.tr.DialTLS = cli.DialTLS
			}
			cli.tr.Proxy = http.ProxyURL(cli.urlproxy)
		} else {
			cli.tr.Proxy = http.ProxyFromEnvironment
		}
	} else {
		cli.qconf = &quic.Config{
			MaxIdleTimeout:                 90 * time.Second,
			HandshakeIdleTimeout:           10 * time.Second,
			KeepAlivePeriod:                30 * time.Second,
			InitialStreamReceiveWindow:     512 * 1024 * 10,
			InitialConnectionReceiveWindow: 512 * 1024 * 10,
		}
		cli.tr3 = &http3.RoundTripper{TLSClientConfig: cli.tlsconfig, QuicConfig: cli.qconf}
	}
	//

	cli.client = &http.Client{}
	if cli.cfg.Http3 == true {
		cli.client.Transport = cli.tr3
	} else {
		cli.client.Transport = cli.tr
	}
	//for cache tcp & dns(not must)
	//res, _ := cli.Dummy_Get(cli.cfg.Fetchserver)
	//if res != nil {
	//	if res.Body != nil {
	//		res.Body.Close()
	//	}
	//}
}

func (cli *client) VerifyConnection(cs tls.ConnectionState) error {
	//
	cert := cs.PeerCertificates[0]
	if reflect.DeepEqual(cert, cli.cert) {
		return errors.New("This is a middle attack server using Php-Proxy CA")
	} else {
		cli.tlsconfig.VerifyConnection = nil
		return nil
	}
}

func (cli *client) DialTLS(network, addr string) (net.Conn, error) {
	tlsconfig := &tls.Config{
		InsecureSkipVerify: cli.cfg.Insecure,
	}
	conn, err := tls.Dial(network, addr, tlsconfig)
	return conn, err
}

//for HTTPS Forward PROXY dialer for feature use(need test)
func (cli *client) Dial(network, addr string) (c net.Conn, err error) {
	tlsconfig := &tls.Config{
		InsecureSkipVerify: cli.cfg.Insecure,
	}
	var port string
	if cli.urlproxy.Port() == "" {
		if cli.urlproxy.Scheme == "http" {
			port = "80"
		} else if cli.urlproxy.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	conn, err := tls.Dial("tcp", cli.urlproxy.Hostname()+":"+port, tlsconfig)
	if err != nil {
		return conn, errors.New("HTTPS Proxy:" + err.Error())
	}
	req := &http.Request{
		Method:        http.MethodConnect,
		Header:        http.Header{},
		Host:          addr,
		URL:           &url.URL{Path: addr},
		ContentLength: 0,
		Body:          nil,
	}
	req.Header.Set("Host", addr)
	req.Header.Set("Proxy-Connection", "keep-alive")
	if cli.urlproxy.User != nil {
		p := base64.StdEncoding.EncodeToString([]byte(cli.urlproxy.User.String()))
		log.Print(p)
		req.Header.Set("Proxy-Authorization", "Basic "+p)
	}
	//
	err_wr := req.Write(conn)
	if err_wr != nil {
		conn.Close()
		return conn, errors.New("HTTPS Proxy:" + err_wr.Error())
	}
	var b [1024]byte
	n, err := conn.Read(b[:])
	if err != nil {
		conn.Close()
		return conn, errors.New("HTTPS Proxy:" + err.Error())
	}
	Res, err_rd := http.ReadResponse(bufio.NewReader(bytes.NewReader(b[:n])), req)
	if err_rd != nil {
		conn.Close()
		return conn, errors.New("HTTPS Proxy:" + err_rd.Error())
	}
	if Res.StatusCode != http.StatusOK {
		conn.Close()
		return conn, errors.New("HTTPS Proxy:" + "Connect status error")
	}
	return conn, nil

}
