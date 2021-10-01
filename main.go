package main

import (
	"flag"
	"fmt"
)

var ip = flag.String("ip", "localhost", "ip addr server binds to")
var port = flag.String("port", "1080", "port number server binds to")

func main() {
	flag.Parse()
	server, er := NewSocksServer(*ip,*port)
	if er != nil {
		fmt.Println("Server Creation Error!")
		fmt.Println(er)
		return
	}
	server.Start()
}
