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

func makeInput(size int) []byte {
	return makeInputI(size, 0)
}

func makeInputI(size int, initialValue byte) []byte {
	result := make([]byte, size)
	for i := range result {
		result[i] = initialValue
	}
	return result
}

func connToChannel(conn net.Conn) <-chan []byte {
	channel := make(chan []byte,1)
	go func() {
		for {
			input := makeInput(1024)
			n ,er := conn.Read(input)
			if er != nil {
				logger.Errorln("Error Read From:" + conn.RemoteAddr().String())
				logger.Errorln(er)
				close(channel)
				return
			} else if n > 0 {
				if *lpackets {
					logger.Infof("%v Bytes Read From %v\n", n, conn.RemoteAddr().String())
				}
				channel <- input[:n]
			}
		}

	}()
	return channel
}


func writeToConnection(conn net.Conn, data []byte, errorMsg string) (int, error) {
	n, er := conn.Write(data)
	if er != nil {
		if *logging {
			logger.Errorln(errorMsg)
			logger.Errorln(conn.RemoteAddr().String())
			logger.Errorln(er)
		}
		return 0, er
	}
	return n, nil
}

func readFromConnection(conn net.Conn, input []byte, errorMsg string) (int, error) {
	n, er := conn.Read(input)
	if er != nil {
		if *logging {
			logger.Errorln(errorMsg)
			logger.Errorln(conn.RemoteAddr().String())
			logger.Errorln(er)
		}
		return 0, er
	}
	return n, nil
}


func createConnectionIpv4(ip uint32, port uint16) (net.Conn, error) {
	address := FormatIpAndPort(ip, port)
	if *logging {
		logger.Infoln(address)
	}
	conn, er := net.Dial("tcp", address)

	if er != nil {
		if *logging {
			logger.Errorln("Error Connecting Destination Ip&Port!")
			logger.Errorln(er)
		}
		return nil, er
	}
	return conn, nil
}

func createConnectionDomain(domain string, port uint16) (net.Conn, error) {
	address := domain + ":" + strconv.Itoa(int(port))
	if *logging {
		logger.Infoln(address)
	}
	conn, er := net.Dial("tcp", address)

	if er != nil {
		if *logging {
			logger.Errorln("Error Connecting Destination DomainName&Port!")
			logger.Errorln(er)
		}
		return nil, er
	}
	return conn, nil
}

func extractNullTerminatedString(conn net.Conn) ([]byte,error) {
	result := makeInput(0)
	input := makeInput(1)
	for {
		_, er := conn.Read(input)
		if er != nil {
			return nil, er
		}
		result = append(result, input[0])
		if input[0] == 0 {
			return result,nil
		}
	}
}

func extractSizePrefixedString(conn net.Conn) ([]byte,error) {
	result := makeInput(0)
	input := makeInput(1)
	_, er := conn.Read(input)
	if er != nil {
		return nil, er
	}
	result = append(result, input[0])
	size := input[0]
	if size > 0 {
		input = makeInput(int(size))
		_, er := conn.Read(input)
		if er != nil {
			return nil, er
		}
		result = append(result, input...)
	}
	return result, nil
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