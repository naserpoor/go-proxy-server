package main

import (
	"encoding/binary"
	"fmt"
	"github.com/google/logger"
	"math"
	"net"
	"strconv"
	"strings"
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

func ParseIpStr(ipStr string) (uint32, error) {
	const errorWrongFormat = "Ip wrong format"
	if strings.ToLower(ipStr) == "localhost" {
		ipStr = "127.0.0.1"
	}
	ipParts := strings.Split(ipStr, ".")
	if len(ipParts) != 4 {
		return 0, errorT{error: errorWrongFormat}
	}
	ipBytes := make([]byte, 4)
	for idx, part := range ipParts {
		res, er := strconv.Atoi(part)
		if er != nil || res > 255 || res < 0 {
			return 0, errorT{error: errorWrongFormat}
		}
		ipBytes[idx] = byte(res)
	}
	return binary.LittleEndian.Uint32(ipBytes), nil
}

func ParsePortStr(portStr string) (uint16, error) {
	port, er := strconv.Atoi(portStr)
	if er != nil {
		return 0, er
	}
	if port > math.MaxUint16 || port < 0 {
		return 0, errorT{"Wrong Port Number:" + portStr}
	}
	return uint16(port), nil
}

func ParseIpUint32(ipUint uint32) string {
	ipBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(ipBytes, ipUint)
	return fmt.Sprintf("%v.%v.%v.%v", ipBytes[0], ipBytes[1], ipBytes[2], ipBytes[3])
}

func FormatIpAndPort(ip uint32, port uint16) string {
	return fmt.Sprintf("%v:%v", ParseIpUint32(ip), port)
}

type errorT struct {
	error string
}

func (a errorT) Error() string {
	return a.error
}
