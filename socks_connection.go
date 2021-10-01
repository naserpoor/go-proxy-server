package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"reflect"
	"strconv"
)

type Connection struct {
	conn net.Conn
	state State
}

func NewConnection(conn net.Conn) Connection {
	return Connection{conn, SocksRecognizerState{conn}}
}

func (c Connection) StartConnectionProcess() {
	for c.state != nil {
		fmt.Println("Current Connection State:")
		fmt.Println(reflect.TypeOf(c.state))
		nextState, er := c.state.ProcessData()
		if er != nil {
			fmt.Println("State Returned Error!")
			fmt.Println(er)
			return
		}
		c.state = nextState
	}
	return
}

type State interface {
	ProcessData() (State,error)
}

type SocksRecognizerState struct {
	conn net.Conn
}
type Socks4InitialState struct {
	conn net.Conn
}
type Socks4ConnectingState struct {
	conn net.Conn
	ip uint32
	port uint16
	userId []byte
}

type SocksDirectingState struct {
	conn net.Conn
	conn2 net.Conn
}

type Socks4aConnectingState struct {
	conn net.Conn
	port uint16
	userId []byte
	domainName []byte
}
type Socks5InitialState struct {
	conn net.Conn
}


func (s SocksRecognizerState) ProcessData() (State,error) {
	input := make([]byte, 1)
	_,err := s.conn.Read(input)
	if err != nil {
		fmt.Println("Error Reading Socks Ver From Connection!")
		fmt.Println(err)
		return nil, err
	}
	socksVer := input[0]
	switch socksVer {
	case 4:
		return Socks4InitialState{s.conn},nil
	case 5:
		return Socks5InitialState{s.conn},nil
	default:
		return nil, errorT{error: "Socks Version Not Recognized:"+ string(socksVer)}
	}
}

func (s Socks4InitialState) ProcessData() (State,error) {
	input := make([]byte, 1)
	_,er := s.conn.Read(input)
	if er != nil {
		fmt.Println("Error Reading Socks4 Command!")
		fmt.Println(er)
		return nil, er
	}
	cmd := input[0]
	if cmd != 1 {
		fmt.Println("Error NotSupported Socks4 Command:" + strconv.Itoa(int(cmd)))
		return nil, errorT{error: "Error NotSupported Socks4 Command:"+strconv.Itoa(int(cmd))}
	}
	fmt.Println("Received Command:" + strconv.Itoa(int(cmd)))
	input = make([]byte,6)
	n,er := s.conn.Read(input)
	if er != nil {
		fmt.Println("Error Reading Socks4 DstIp$DstPort")
		fmt.Println(er)
		return nil, er
	}
	if n != 6 {
		fmt.Println("Error Not Enough Data For DstIp&DstPort:"+strconv.Itoa(n))
		return nil,errorT{"Error Not Enough Data For DstIp&DstPort:"+strconv.Itoa(n)}
	}
	fmt.Printf("Received IP&PORT:%v\n", input)
	ip := binary.LittleEndian.Uint32(input[2:6])
	port := binary.BigEndian.Uint16(input[0:2])

	userId := makeInput(0)
	input = makeInput(1)
	for {
		_,er := s.conn.Read(input)
		if er != nil {
			fmt.Println("Error Reading Socks4 UserId")
			fmt.Println(er)
			return nil, er
		}
		if input[0] != 0 {
			userId = append(userId, input[0])
		} else {
			break
		}
	}
	fmt.Printf("Received UserID:%v\n", userId)

	if ip & 0x000000ff == ip {
		domainName := makeInput(0)
		for {
			_,er := s.conn.Read(input)
			if er != nil {
				fmt.Println("Error Reading Socks4a DomainName")
				fmt.Println(er)
				return nil, er
			}
			if input[0] != 0 {
				domainName = append(domainName, input[0])
			} else {
				break
			}
		}
		fmt.Printf("Received UserID:%v\n", domainName)
		return Socks4aConnectingState{s.conn,port,userId,domainName}, nil
	} else {
		return Socks4ConnectingState{s.conn,ip,port,userId}, nil
	}
}

func (s Socks4ConnectingState) ProcessData() (State, error) {
	address := FormatIpAndPort(s.ip, s.port)
	fmt.Println(address)
	conn2, er := net.Dial("tcp", address)
	if er != nil {
		fmt.Println("Error Connecting Destination Ip&Port!")
		fmt.Println(er)
		s.conn.Write(socks4ReplyMessage(0x5B,0,0))
		return nil, er
	}
	n, er := s.conn.Write(socks4ReplyMessage(0x5A,0,0))
	if er != nil || n < 8 {
		fmt.Println("Error Sending Socks Reply")
		fmt.Println(er)
		return nil, er
	}
	return SocksDirectingState{s.conn,conn2} ,nil
}

func (s Socks4aConnectingState) ProcessData() (State, error) {
	address := string(s.domainName) + strconv.Itoa(int(s.port))
	fmt.Println(address)
	conn2, er := net.Dial("tcp", address)
	if er != nil {
		fmt.Println("Error Connecting Destination DomainName&Port!")
		fmt.Println(er)
		s.conn.Write(socks4ReplyMessage(0x5B,0,0))
		return nil, er
	}
	n, er := s.conn.Write(socks4ReplyMessage(0x5A,0,0))
	if er != nil || n < 8 {
		fmt.Println("Error Sending Socks Reply")
		fmt.Println(er)
		return nil, er
	}
	return SocksDirectingState{s.conn,conn2} ,nil
}

func (s SocksDirectingState) ProcessData() (State, error) {
	chan1 := connToChannel(s.conn)
	chan2 := connToChannel(s.conn2)

	var input []byte
	for{
		select {
		case input = <- chan1:
			_,err := s.conn2.Write(input)
			if err != nil {
				fmt.Println("Error Directing Socks Write!:" + s.conn2.RemoteAddr().String())
				fmt.Println(err)
				return nil,err
			}
		case input = <- chan2:
			_,err := s.conn.Write(input)
			if err != nil {
				fmt.Println("Error Directing Socks Write!:" + s.conn.RemoteAddr().String())
				fmt.Println(err)
				return nil,err
			}
		}
	}
}

func (s Socks5InitialState) ProcessData() (State, error) {
	return nil, nil
}

func socks4ReplyMessage(code byte, ip uint32, port uint16) []byte {
	result := []byte{0,code}
	ipBytes :=  makeInput(4)
	portBytes := makeInput(2)
	binary.LittleEndian.PutUint16(portBytes,port)
	binary.LittleEndian.PutUint32(ipBytes,ip)
	result = append(result, portBytes...)
	result = append(result, ipBytes...)
	fmt.Printf("Reply Bytes:%v\n", result)
	return result
}