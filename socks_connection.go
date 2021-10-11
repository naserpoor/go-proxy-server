package main

import (
	"encoding/binary"
	"github.com/google/logger"
	"net"
	"reflect"
	"strconv"
)

type Connection struct {
	conn  net.Conn
	state State
}

func NewConnection(conn net.Conn) Connection {
	return Connection{conn, SocksRecognizerState{conn}}
}

func (c Connection) StartConnectionProcess() {
	for c.state != nil {
		if *logging {
			logger.Infoln("Current Connection State:")
			logger.Infoln(reflect.TypeOf(c.state))
		}
		nextState, er := c.state.ProcessData()
		if er != nil {
			if *logging {
				logger.Errorln("State Returned Error!")
				logger.Errorln(er)
			}
			c.conn.Close()
			return
		}
		c.state = nextState
	}
	return
}

type State interface {
	ProcessData() (State, error)
}

type SocksRecognizerState struct {
	conn net.Conn
}
type Socks4InitialState struct {
	conn net.Conn
}
type SocksIpv4PortConnectingState struct {
	conn   net.Conn
	ip     uint32
	port   uint16
	userId []byte
	socksVer byte
	bytes []byte
}

type SocksDirectingState struct {
	conn  net.Conn
	conn2 net.Conn
}

type SocksDomainPortConnectingState struct {
	conn       net.Conn
	port       uint16
	userId     []byte
	domainName []byte
	socksVer byte
	bytes []byte
}
type Socks5InitialState struct {
	conn net.Conn
}

type Socks5UserPasswordAuth struct {
	conn       net.Conn
}

type Socks5ConnectingState struct {
	conn       net.Conn
}

func (s SocksRecognizerState) ProcessData() (State, error) {
	input := make([]byte, 1)
	_, err := s.conn.Read(input)
	if err != nil {
		if *logging {
			logger.Errorln("Error Reading Socks Ver From Connection!")
			logger.Errorln(err)
		}
		return nil, err
	}
	socksVer := input[0]
	switch socksVer {
	case 4:
		return Socks4InitialState{s.conn}, nil
	case 5:
		return Socks5InitialState{s.conn}, nil
	default:
		return nil, errorT{error: "Socks Version Not Recognized:" + string(socksVer)}
	}
}

func (s Socks4InitialState) ProcessData() (State, error) {
	input := make([]byte, 1)
	_, er := s.conn.Read(input)
	if er != nil {
		if *logging {
			logger.Errorln("Error Reading Socks4 Command!")
			logger.Errorln(er)
		}
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
		logger.Infof("Received Command: %v\n" , cmd)
	}
	input = make([]byte, 6)
	n, er := s.conn.Read(input)
	if er != nil {
		if *logging {
			logger.Errorln("Error Reading Socks4 DstIp$DstPort")
			logger.Errorln(er)
		}
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
		_, er := s.conn.Read(input)
		if er != nil {
			if *logging {
				logger.Errorln("Error Reading Socks4 UserId")
				logger.Errorln(er)
			}
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
		domainName := makeInput(0)
		for {
			_, er := s.conn.Read(input)
			if er != nil {
				if *logging {
					logger.Errorln("Error Reading Socks4a DomainName")
					logger.Errorln(er)
				}
				return nil, er
			}
			if input[0] != 0 {
				domainName = append(domainName, input[0])
			} else {
				break
			}
		}
		if *logging {
			logger.Infof("Received DomainName: %v\n", domainName)
		}
		return SocksDomainPortConnectingState{ s.conn, port, userId, domainName, 0x04, nil}, nil
	} else {
		return SocksIpv4PortConnectingState{s.conn, ip, port, userId, 0x04, nil}, nil
	}
}

func (s SocksIpv4PortConnectingState) ProcessData() (State, error) {
	address := FormatIpAndPort(s.ip, s.port)
	if *logging {
		logger.Infoln(address)
	}
	conn2, er := net.Dial("tcp", address)
	return processSecondConnections(s.socksVer, s.conn, conn2, er, false,s.bytes)
}

func (s SocksDomainPortConnectingState) ProcessData() (State, error) {
	address := string(s.domainName) + ":" + strconv.Itoa(int(s.port))
	if *logging {
		logger.Infoln(address)
	}
	conn2, er := net.Dial("tcp", address)
	return processSecondConnections(s.socksVer, s.conn, conn2, er, true,s.bytes)
}

func (s SocksDirectingState) ProcessData() (State, error) {
	chan1 := connToChannel(s.conn)
	chan2 := connToChannel(s.conn2)

	for {
		select {
		case input, ok := <-chan1:
			er := writeToConn(s.conn2, input, ok)
			if er != nil {
				return nil, er
			}
		case input, ok := <-chan2:
			er := writeToConn(s.conn, input, ok)
			if er != nil {
				return nil, er
			}
		}
	}
}

func (s Socks5InitialState) ProcessData() (State, error) {
	input := makeInput(1)
	_, er := s.conn.Read(input)
	if er != nil {
		if *logging {
			logger.Errorln("Error Reading Socks5 nAuth!")
			logger.Errorln(er)
		}
		return nil, er
	}

	nAuth := input[0]
	if nAuth <= 0 {
		return nil,errorT{error:"nAuth is Less Equal Than 0!" }
	}
	input = makeInput(int(nAuth))
	_, er = s.conn.Read(input)
	if er != nil {
		if *logging {
			logger.Errorln("Error Reading Socks5 auths!")
			logger.Errorln(er)
		}
		return nil, er
	}
	auths := input
	hasValidAuth := false
	chosenAuth := byte(0x00)
	for _, it := range auths {
		if it == 0x00 || it == 0x02 {
			hasValidAuth = true
			chosenAuth = it
		}
	}
	if !hasValidAuth {
		s.conn.Write([]byte{0x05,0xFF})
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
		return Socks5ConnectingState{conn: s.conn}, nil

	case 0x02:
		return Socks5UserPasswordAuth{conn: s.conn}, nil
	}
	return nil, nil
}

func (s Socks5ConnectingState) ProcessData() (State, error) {
	input := makeInput(3)
	_, er := s.conn.Read(input)

	if er != nil {
		if *logging {
			logger.Errorln("Error Socks5 Reading Connecting Cmd!")
			logger.Errorln(er)
		}
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

	switch result.addrType {
	case 0x01:
		return SocksIpv4PortConnectingState{conn: s.conn, port: result.port, ip: result.ipv4, socksVer: 0x05, bytes: result.bytes}, nil

	default:
		return nil, errorT{"Not Supported Socks5 Address Type"}
	}
}

func (s Socks5UserPasswordAuth) ProcessData() (State, error) {
	s.conn.Close()
	return nil, nil
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

func processSecondConnections(socksVer byte, conn1 net.Conn, conn2 net.Conn, er error, domain bool, bytes []byte) (State, error) {
	if er != nil {
		if *logging {
			if domain {
				logger.Errorln("Error Connecting Destination DomainName&Port!")
			} else {
				logger.Errorln("Error Connecting Destination Ip&Port!")
			}
			logger.Errorln(er)
		}
		if socksVer == 0x04 {
			conn1.Write(socks4ReplyMessage(0x5B, 0, 0))
		} else {
			conn1.Write(socks5ReplyMessage(0x05, bytes))
		}
		return nil, er
	}
	if socksVer == 0x04 {
		_, er = conn1.Write(socks4ReplyMessage(0x5A, 0, 0))
	} else {
		_, er = conn1.Write(socks5ReplyMessage(0x00, bytes))
	}

	if er != nil{
		if *logging {
			logger.Errorln("Error Sending Socks Reply")
			logger.Errorln(er)
		}
		return nil, er
	}
	return SocksDirectingState{conn1, conn2}, nil
}

func writeToConn(conn net.Conn, input []byte, ok bool) error {
	if !ok {
		err := conn.Close()
		if err != nil {
			return err
		}
		return errorT{"Error Socks Channel Closed!"}
	}
	n, err := conn.Write(input)
	if *lpackets && *logging {
		logger.Infof("%v Bytes Written To %v\n", n, conn.RemoteAddr().String())
	}
	if err != nil {
		if *logging {
			logger.Errorln("Error Directing Socks Write!:" + conn.RemoteAddr().String())
			logger.Errorln(err)
		}
		return err
	}
	return nil
}

type Socks5Address struct {
	addrType byte
	ipv4 uint32
	domainName string
	port uint16
	bytes []byte
}

func readSocks5Address(conn net.Conn) (*Socks5Address, error) {
	result := new(Socks5Address)
	result.bytes = makeInput(0)
	input := makeInput(1)
	_, er := conn.Read(input)
	if er != nil {
		if *logging {
			logger.Errorln("Error Reading Socks5 Address Type!")
			logger.Errorln(er)
		}
		return nil, er
	}
	result.bytes = append(result.bytes, input...)
	addrType := input[0]
	result.addrType = addrType

	switch addrType {
	case 0x01:
		input = makeInput(4)
		_, er := conn.Read(input)
		if er != nil {
			if *logging {
				logger.Errorln("Error Reading Socks5 Ip Address!")
				logger.Errorln(er)
			}
			return nil, er
		}

		result.bytes = append(result.bytes, input...)
		ipv4 := binary.LittleEndian.Uint32(input)
		result.ipv4 = ipv4

	case 0x03:


	case 0x04:
	}

	input = makeInput(2)
	_, er = conn.Read(input)
	if er != nil {
		if *logging {
			logger.Errorln("Error Socks5 Reading Port Address!")
			logger.Errorln(er)
		}
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
