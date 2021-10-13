package main

import (
	"fmt"
	"github.com/google/logger"
	"net"
)

type SocksServer struct {
	ip   uint32
	port uint16
}

func NewSocksServer(ipStr string, portStr string) (result *SocksServer, err error) {
	ip, er := ParseIpStr(ipStr)
	if er != nil {
		return nil, er
	}
	port, er := ParsePortStr(portStr)
	if er != nil {
		return nil, er
	}
	server := SocksServer{ip: ip, port: port}
	return &server, nil
}

func (server *SocksServer) Start() {
	ser, er := net.Listen("tcp", fmt.Sprintf("%v:%v", ParseIpUint32(server.ip), server.port))
	if er != nil {
		if *logging {
			logger.Errorln("Server Cant Start")
			logger.Errorln(er)
		}
		return
	}
	logger.Infoln("Server Start Accepting...")
	for {
		conn, er := ser.Accept()
		if er != nil {
			if *logging {
				logger.Errorln("Server Error Accepting!")
				logger.Errorln(er)
			}
			return
		}
		if *logging {
			logger.Infoln("New Connection Accepted!")
		}
		connection := NewConnection(conn)
		go connection.StartConnectionProcess()
	}
}
