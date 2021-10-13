package main

import (
	"encoding/binary"
	"github.com/google/logger"
	"net"
)

type Socks5InitialState struct {
	conn net.Conn
}

type Socks5UserPasswordAuth struct {
	conn net.Conn
}

type Socks5CommandState struct {
	conn net.Conn
}

type Socks5ConnectingState struct {
	conn net.Conn
	socks4Addr Socks5Address
}


func (s Socks5InitialState) ProcessData() (State, error) {
	input := makeInput(1)
	_, er := readFromConnection(s.conn, input, "Error Reading Socks5 nAuth!")
	if er != nil {
		return nil, er
	}

	nAuth := input[0]
	if nAuth <= 0 {
		return nil, errorT{error: "nAuth is Less Equal Than 0!"}
	}
	input = makeInput(int(nAuth))
	_, er = readFromConnection(s.conn, input, "Error Reading Socks5 auths!")
	if er != nil {
		return nil, er
	}
	auths := input
	hasValidAuth := false
	chosenAuth := byte(0x00)
	for _, it := range auths {
		if it == 0x02 {
			hasValidAuth = true
			chosenAuth = it
		}
	}
	if !hasValidAuth {
		s.conn.Write([]byte{0x05, 0xFF})
		return nil, errorT{"No Valid Auth Found!"}
	}

	_, er = s.conn.Write([]byte{0x05, chosenAuth})
	if er != nil {
		if *logging {
			logger.Errorln("Send Socks5 Chosen Auth Error!")
			logger.Errorln(er)
		}
		return nil, er
	}

	switch chosenAuth {
	case 0x00:
		return Socks5CommandState{conn: s.conn}, nil

	case 0x02:
		return Socks5UserPasswordAuth{conn: s.conn}, nil
	}
	return nil, nil
}

func (s Socks5CommandState) ProcessData() (State, error) {
	input := makeInput(3)
	_, er := readFromConnection(s.conn, input, "Error Socks5 Reading Connecting Cmd!")
	if er != nil {
		return nil, er
	}

	if input[0] != 0x05 || input[2] != 0x00 {
		return nil, errorT{"Error Socks5 Connecting Request Type!"}
	}

	cmd := input[1]
	if cmd != 0x01 {
		return nil, errorT{"Error Socks5 Not Supported Command"}
	}

	result, er := readSocks5Address(s.conn)
	if er != nil {
		return nil, er
	}

	return Socks5ConnectingState{s.conn,*result}, nil
}

func (s Socks5UserPasswordAuth) ProcessData() (State, error) {
	input := makeInput(1)
	_, er := readFromConnection(s.conn, input, "Error Socks5 Reading User Password Version")
	if er != nil {
		return nil, er
	}

	_, er = readFromConnection(s.conn, input, "Error Socks5 Reading Username Length")
	if er != nil {
		return nil, er
	}

	user_name := ""
	if input[0] > 0 {
		input = makeInput(int(input[0]))
		_, er = readFromConnection(s.conn, input, "Error Socks5 Reading Username")
		if er != nil {
			return nil, er
		}
		user_name = string(input)
	}

	input = makeInput(1)
	_, er = readFromConnection(s.conn, input, "Error Socks5 Reading Password Length")
	if er != nil {
		return nil, er
	}

	pass_word := ""
	if input[0] > 0 {
		input = makeInput(int(input[0]))
		_, er = readFromConnection(s.conn, input, "Error Socks5 Reading Username")
		if er != nil {
			return nil, er
		}
		pass_word = string(input)
	}

	if user_name != "Alirexa" || pass_word != "Alirexa" {
		writeToConnection(s.conn, []byte{0x01, 0x01}, "")
		return nil, errorT{ "Error Wrong Username Or Password:" + user_name + ":" + pass_word }
	}

	_, er = writeToConnection(s.conn, []byte{0x01, 0x00}, "Error Writing User Password Auth Reply")
	if er != nil {
		return nil, er
	}

	return Socks5CommandState{s.conn}, nil
}

func (s Socks5ConnectingState) ProcessData() (State, error) {
	var (
		conn2 net.Conn
		er error
	)
	switch s.socks4Addr.addrType {
	case 0x01:
		conn2, er = createConnectionIpv4(s.socks4Addr.ipv4, s.socks4Addr.port)
	case 0x03:
		conn2, er = createConnectionDomain(s.socks4Addr.domainName,s.socks4Addr.port)
	default:
		return nil, errorT{"Not Supported Socks5 Address Type"}
	}
	if er != nil {
		writeToConnection(s.conn, socks5ReplyMessage(0x05, s.socks4Addr.bytes), "Error Writing Socks5 Reply")
		return nil, er
	}
	_, er = writeToConnection(s.conn, socks5ReplyMessage(0x00, s.socks4Addr.bytes), "Error Writing Socks5 Reply")
	if er != nil {
		return nil, er
	}
	return SocksDirectingState{s.conn, conn2}, nil
}




type Socks5Address struct {
	addrType   byte
	ipv4       uint32
	domainName string
	port       uint16
	bytes      []byte
}

func readSocks5Address(conn net.Conn) (*Socks5Address, error) {
	result := new(Socks5Address)
	result.bytes = makeInput(0)
	input := makeInput(1)
	_, er := readFromConnection(conn, input, "Error Reading Socks5 Address Type!")
	if er != nil {
		return nil, er
	}
	result.bytes = append(result.bytes, input...)
	addrType := input[0]
	result.addrType = addrType

	switch addrType {
	case 0x01:
		input = makeInput(4)
		_, er := readFromConnection(conn, input, "Error Reading Socks5 Ip Address!")
		if er != nil {
			return nil, er
		}

		result.bytes = append(result.bytes, input...)
		ipv4 := binary.LittleEndian.Uint32(input)
		result.ipv4 = ipv4

	case 0x03:
		addressBytes, er := extractSizePrefixedString(conn)
		if er != nil {
			return nil, er
		}
		result.domainName = string(addressBytes[1:])
		result.bytes = append(result.bytes, addressBytes...)

	case 0x04:
		return nil, errorT{"Not Implemented!"}
	}

	input = makeInput(2)
	_, er = readFromConnection(conn, input, "Error Socks5 Reading Port Address!")
	if er != nil {
		return nil, er
	}
	result.bytes = append(result.bytes, input...)
	port := binary.BigEndian.Uint16(input)
	result.port = port

	return result, nil
}

func socks5ReplyMessage(status byte, socks5Addr []byte) []byte {
	return append([]byte{0x05, status, 0x00}, socks5Addr...)
}