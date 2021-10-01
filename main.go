package main

import (
	"flag"
	"fmt"
	"github.com/google/logger"
	"log"
	"os"
)

var ip = flag.String("ip", "0.0.0.0", "ip addr server binds to")
var port = flag.String("port", "1080", "port number server binds to")
var logging = flag.Bool("logging",true, "enables logging")
var verbose = flag.Bool("verbose",false,"show info logs on terminal")
var lpackets = flag.Bool("lpackets",false,"logs every packet (heavy log file)")

func main() {
	flag.Parse()
	if *logging {
		initLogger()
		defer logger.Close()
	}

	server, er := NewSocksServer(*ip,*port)
	if er != nil {
		if *logging {
			logger.Errorln("Server Creation Error!")
			logger.Errorln(er)
		}
		return
	}
	server.Start()
}

func initLogger() {
	file,er := os.OpenFile("./logs.txt",os.O_CREATE|os.O_WRONLY|os.O_APPEND,0777)
	if er != nil {
		fmt.Println("Error Opening Log File")
	}
	logger.Init("Log",*verbose,true,file)
	logger.SetFlags(log.LstdFlags|log.Lmicroseconds)
}
