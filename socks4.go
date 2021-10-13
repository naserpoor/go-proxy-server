package main

import (
	"encoding/binary"
	"github.com/google/logger"
	"net"
	"strconv"
)

type Socks4InitialState struct {
	conn net.Conn
}

type Socks4ConnectingState struct {
	conn   net.Conn
	ip     uint32
	port   uint16
	userId []byte
}

type Socks4aConnectingState struct {
	conn       net.Conn
	port       uint16
	userId     []byte
	domainName string
}

type Socks4ConnectedState struct {
	conn  net.Conn
	conn2 net.Conn
}


func (s Socks4InitialState) ProcessData() (State, error) {
	input := make([]byte, 1)
	_, er := readFromConnection(s.conn, input, "Error Reading Socks4 Command!")
	if er != nil {
		return nil, er
	}
	cmd := input[0]
	if cmd != 1 {
		if *logging {
			logger.Errorln("Error NotSupported Socks4 Command:" + strconv.Itoa(int(cmd)))
		}
		return nil, errorT{error: "Error NotSupported Socks4 Command:" + strconv.Itoa(int(cmd))}
	}
	if *logging {
		logger.Infof("Received Command: %v\n", cmd)
	}
	input = make([]byte, 6)
	n, er := readFromConnection(s.conn, input, "Error Reading Socks4 DstIp$DstPort")
	if er != nil {
		return nil, er
	}
	if n != 6 {
		if *logging {
			logger.Errorln("Error Not Enough Data For DstIp&DstPort:" + strconv.Itoa(n))
		}
		return nil, errorT{"Error Not Enough Data For DstIp&DstPort:" + strconv.Itoa(n)}
	}
	if *logging {
		logger.Infof("Received IP&PORT:%v\n", input)
	}
	ip := binary.LittleEndian.Uint32(input[2:6])
	port := binary.BigEndian.Uint16(input[0:2])

	userId := makeInput(0)
	input = makeInput(1)
	for {
		_, er := readFromConnection(s.conn, input, "Error Reading Socks4 UserId")
		if er != nil {
			return nil, er
		}
		if input[0] != 0 {
			userId = append(userId, input[0])
		} else {
			break
		}
	}
	if *logging {
		logger.Infof("Received UserID:%v\n", userId)
	}

	if ip&0x000000ff == ip {
		domainName, er := extractNullTerminatedString(s.conn)
		if er != nil {
			return nil, er
		}
		if *logging {
			logger.Infof("Received DomainName: %v\n", domainName)
		}
		return Socks4aConnectingState{s.conn, port, userId, string(domainName)}, nil
	} else {
		return Socks4ConnectingState{s.conn, ip, port, userId}, nil
	}
}

func (s Socks4ConnectingState) ProcessData() (State, error) {
	conn2, er := createConnectionIpv4(s.ip, s.port)
	if er != nil {
		writeToConnection(s.conn, socks4ReplyMessage(0x5B, 0, 0), "")
		return nil, er
	}

	return Socks4ConnectedState{s.conn, conn2}, nil
}

func (s Socks4aConnectingState) ProcessData() (State, error) {
	conn2, er := createConnectionDomain(s.domainName, s.port)
	if er != nil {
		writeToConnection(s.conn, socks4ReplyMessage(0x5B, 0, 0), "")
		return nil, er
	}
	return Socks4ConnectedState{s.conn, conn2}, nil
}

func (s Socks4ConnectedState) ProcessData() (State, error) {
	_, er := writeToConnection(s.conn, socks4ReplyMessage(0x5A, 0, 0), "Error Sending Socks Reply")
	if er != nil {
		return nil, er
	}
	return SocksDirectingState{s.conn, s.conn2}, nil
}





func socks4ReplyMessage(code byte, ip uint32, port uint16) []byte {
	result := []byte{0, code}
	ipBytes := makeInput(4)
	portBytes := makeInput(2)
	binary.LittleEndian.PutUint16(portBytes, port)
	binary.LittleEndian.PutUint32(ipBytes, ip)
	result = append(result, portBytes...)
	result = append(result, ipBytes...)
	if *logging {
		logger.Infof("Reply Bytes:%v\n", result)
	}
	return result
}