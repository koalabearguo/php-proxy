package main

import (
	"log"
	"os"
)

func main() {
	//
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	//
	config := &config{}
	config.init_config()
	//
	prx := &proxy{cfg: config}
	prx.init_proxy()
	//
}
