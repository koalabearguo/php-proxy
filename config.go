package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
)

//CaCert/CA.crt should be trusted by local OS
//Php-Proxy CA
var CaCert = []byte(`-----BEGIN CERTIFICATE-----
MIIDXjCCAkYCAQAwDQYJKoZIhvcNAQELBQAwdTELMAkGA1UEBhMCQ04xETAPBgNV
BAgMCEludGVybmV0MQ8wDQYDVQQHDAZDZXJuZXQxEjAQBgNVBAoMCVBocC1Qcm94
eTEXMBUGA1UECwwOUGhwLVByb3h5IFJvb3QxFTATBgNVBAMMDFBocC1Qcm94eSBD
QTAeFw0yMTA2MjMwNzA2MTVaFw0zMTA2MjMwNzA2MTVaMHUxCzAJBgNVBAYTAkNO
MREwDwYDVQQIDAhJbnRlcm5ldDEPMA0GA1UEBwwGQ2VybmV0MRIwEAYDVQQKDAlQ
aHAtUHJveHkxFzAVBgNVBAsMDlBocC1Qcm94eSBSb290MRUwEwYDVQQDDAxQaHAt
UHJveHkgQ0EwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDKgDz0kGfq
2lqDUkU07eQZZqrXBQ8gHQqKXTRI5b8hcHWuIsyj+XdS1zqToEMdG6B7Krfg1AMQ
XgBW3cZvuDEKH/NsWc9oNP1PmL1Aa0iziGn4v78uh8LXVZBX3F/kb2/ZzvklhuMy
GnjXB9AfaP/Me+MDstY0T8NcetTdM4FWGoTxZhcR7W45FqDXexVZSJMYa7dQLMRm
zkfu1naY+BJ11eut4nti1jLwOF4DgWxiEPUAr/GPYyukSsuLL8XzouCKYG4BDTUA
dxw8Gu3Jj3bwkEFo8Kn74UKaip/6GkC83ViICCfLRo8iOxpU9ez54SmKojhFMy5h
J96T+XwlNzE3AgMBAAEwDQYJKoZIhvcNAQELBQADggEBAHTE/hzWuT8pS4OJhwEa
Qsv1lDWPALY0jt/RLHD9qKD/Yogwu43HBEV6zoPVTKhH+dXFIhSwqj5OLOkuk9rA
V0SRv10q/PAAgRqbZy8AGjmcQeZoEEoJO3Wi8h1EGv6M5YWnd+dY5+aBsPjPBWBB
kBFIlYB4Go2ShmSyFXK5LX1SoDA0PI6ASDgkueyArt4lVusjcD6VYvchs/cVytDb
d5Gk6asuC0cGn1YRO+tRzCd3/1mkHAIZDbuv2CPy2ylR23dY5Q+rzR+xhO8hOcn3
kYBBO+G91Jv3U0vRHDXHNvp9+CmJcS1CTBh/KeyWqdK+yi1pSVyB8fUwPN0qdoWy
VSc=
-----END CERTIFICATE-----`)

var CaKey = []byte(`-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDKgDz0kGfq2lqD
UkU07eQZZqrXBQ8gHQqKXTRI5b8hcHWuIsyj+XdS1zqToEMdG6B7Krfg1AMQXgBW
3cZvuDEKH/NsWc9oNP1PmL1Aa0iziGn4v78uh8LXVZBX3F/kb2/ZzvklhuMyGnjX
B9AfaP/Me+MDstY0T8NcetTdM4FWGoTxZhcR7W45FqDXexVZSJMYa7dQLMRmzkfu
1naY+BJ11eut4nti1jLwOF4DgWxiEPUAr/GPYyukSsuLL8XzouCKYG4BDTUAdxw8
Gu3Jj3bwkEFo8Kn74UKaip/6GkC83ViICCfLRo8iOxpU9ez54SmKojhFMy5hJ96T
+XwlNzE3AgMBAAECggEAfTX7+tDLoJjxPKADMO4ji11DJ372UkoCuXlWGfkNTJTn
/wt/c6iOEogIrT18IiRx/5Zzai5N0rH9DblFuNCwae1Fq+qAZ5PUSYJNCucLZg9k
Ty3o/dFuNY2vmdQm6u3IwGnM/lpAYzuhGny3QKTA/mRgA2pyLphfWPCObFQrldvo
D8/HkPIrWP7eeRhDFBuYH2Y8evBW+JJ0xKWWyxnwpYnOvfQP/bb38M9weSmB6oC+
XHnWruCldNLrGwVZsgiuvoh/zIXOQlXRz7U9dT/VJZQGGPdiC49rSN8Q5/qKkPST
1sc2ujmbJpHRtjKtCdBLD/yHNVjbF1cVk2AmPkyGqQKBgQDu+r+7sCruZ+q0ixEm
v7YoMIPouZuuv9wePQL5kwmLqmhZq0vyObcSCRfg5P9oONDB53WNmDjvPEEImQbU
EPmvGofbGVaaTYnbAXmZA50IZElrsPfwZdvRBgMB6gVuPPuHmfStWo+bOEsTml0u
wsa7obEb5giDzjzP22tt2mil/QKBgQDY7GKTP5WUqdgp+1gig5IMu4EpoidrmdWr
U7hKB/R6p+ihrCX1xu/TdZAb7bGFya1O60LsnnQ3iF/heNTWOBUYgDGia6eaEpc1
/K5zis055WoEu5hAebqgr+0VJUEiqGuBUnpqyHLXr8LHf3g7cSU5/4KbCu+1paXu
2/9MMY7AQwKBgFlIu3t+3PtHPcwILPdCJucrAQ1g0wZdzfpKJyNhSO6yUtw1gGFW
KMyHMzGlvLqOh4f6VtP47ESNSWrR6Vgvo2lFSz6TX+S0VW3KRkjhrbil5zxh2LAr
Dg4w5czARxkhlYPbBCwEKqT+SiZfxLKkuKT/SvE2ZzX/Rn8N5jwbnn9tAoGAcVeZ
7fxEKOhRxSXKGEaM0lBKnblXRZacmSdmXHAposkG+Sqcrv3iI6gCw0UAA7qr7ldo
oX/tk3KTPplHBCNLioC47ne3m/5oudGsSTzWHJEtQwnN9KpmBD3H78uGbBh6C5lP
02mm7+GrMVf+N3jYDaTe1inxtAS4XcTfcS1XvEcCgYEAiohDWwiDO/GQloKPuZWc
gVEJoAty/LbU/TyW+i3bM94rcJLKY8ySPBmTQb+ifeqvSRo4W2zv3tQ5lBag77y3
pRYFQzk5UG6yo4V0/oo2UUSDY0UoxX9lVNcJvYVwPcNwKHnX9a9fRrlf3o+c5rrE
jIxm1tgZheqRxqpv1LwQ4hQ=
-----END PRIVATE KEY-----`)

//
const version string = "1.1.2"

//

type config struct {
	//php fetchserver path
	Fetchserver string `json:"fetchserver"`
	//password
	Password string `json:"password"`
	//when connect https php server,TLS sni extension
	Sni string `json:"sni"`
	//local listen address
	Listen string `json:"listen"`
	//debug enable
	Debug bool `json:"debug"`
	//insecure connect to php server
	Insecure bool `json:"insecure"`
}

func (c *config) init_config() {
	//
	log.Printf("Php-Proxy version:v%s\n", version)
	//
	flag.CommandLine.SetOutput(os.Stdout)
	//
	flag.StringVar(&c.Listen, "l", "127.0.0.1:8081", "Local listen address(HTTP Proxy address)")
	flag.StringVar(&c.Password, "p", "123456", "php server password")
	flag.StringVar(&c.Sni, "sni", "", "HTTPS sni extension ServerName(default fetchserver hostname)")
	flag.StringVar(&c.Fetchserver, "s", "https://a.bc.com/php-proxy/index.php", "php fetchserver path(http/https)")
	flag.BoolVar(&c.Debug, "d", false, "enable debug mode for debug")
	flag.BoolVar(&c.Insecure, "k", false, "insecure connect to php server(ignore certs verify/middle attack)")
	flag.Parse()
	//c.writeconfig()
	if len(os.Args) < 2 {
		c.loadconfig()
	} else {
		c.writeconfig()
	}
	//
	log.Printf("php Fetch server:%s\n", c.Fetchserver)
}

func (c *config) loadconfig() {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	raw, err1 := ioutil.ReadFile(dir + "/php-proxy.json")
	if err1 != nil {
		//log.Print(err1)
		return
	}
	log.Print("Load config from ./php-proxy.json file")
	err = json.Unmarshal(raw, c)
	if err != nil {
		log.Print(err)
	}
}
func (c *config) writeconfig() {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	raw, err1 := json.MarshalIndent(c, "", "")
	//raw, err1 := json.Marshal(c)
	//log.Print(string(raw))
	if err1 != nil {
		log.Print(err1)
		return
	}
	err = ioutil.WriteFile(dir+"/php-proxy.json", raw, 0644)
	if err != nil {
		log.Print(err)
		return
	}
	log.Print("Write config to ./php-proxy.json file")
}
