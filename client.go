package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
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
	tr *http.Transport
	//tls.Config when connect https server
	tlsconfig *tls.Config
	//http.Client used to connect server
	client *http.Client
	//ca root cert info for middle attack check
	cert *x509.Certificate
	//server
	server *url.URL
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
	//tr http.client default tr + tlsconfig
	cli.tr = &http.Transport{
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
		TLSClientConfig:       cli.tlsconfig,
	}
	//
	cli.client = &http.Client{
		Transport: cli.tr,
	}
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
