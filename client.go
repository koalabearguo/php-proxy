package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
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
}

func (cli *client) Post(url, contentType string, body io.Reader) (resp *http.Response, err error) {
	return cli.client.Post(url, contentType, body)
}

func (cli *client) init_client() {
	//tls config
	cli.tlsconfig = &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: cli.cfg.insecure,
		VerifyConnection:   cli.VerifyConnection,
	}
	if cli.cfg.insecure == true {
		cli.tlsconfig.VerifyConnection = nil
	}
	if cli.cfg.sni != "" {
		cli.tlsconfig.ServerName = cli.cfg.sni
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
