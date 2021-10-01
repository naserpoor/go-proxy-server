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
type Socks4ConnectingState struct {
	conn   net.Conn
	ip     uint32
	port   uint16
	userId []byte
}

type SocksDirectingState struct {
	conn  net.Conn
	conn2 net.Conn
}

type Socks4aConnectingState struct {
	conn       net.Conn
	port       uint16
	userId     []byte
	domainName []byte
}
type Socks5InitialState struct {
	conn net.Conn
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
		return Socks4aConnectingState{s.conn, port, userId, domainName}, nil
	} else {
		return Socks4ConnectingState{s.conn, ip, port, userId}, nil
	}
}

func (s Socks4ConnectingState) ProcessData() (State, error) {
	address := FormatIpAndPort(s.ip, s.port)
	if *logging {
		logger.Infoln(address)
	}
	conn2, er := net.Dial("tcp", address)
	return processSecondConnections(s.conn, conn2, er, false)
}

func (s Socks4aConnectingState) ProcessData() (State, error) {
	address := string(s.domainName) + ":" + strconv.Itoa(int(s.port))
	if *logging {
		logger.Infoln(address)
	}
	conn2, er := net.Dial("tcp", address)
	return processSecondConnections(s.conn, conn2, er, true)
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

func processSecondConnections(conn1 net.Conn, conn2 net.Conn, er error, domain bool) (State, error) {
	if er != nil {
		if *logging {
			if domain {
				logger.Errorln("Error Connecting Destination DomainName&Port!")
			} else {
				logger.Errorln("Error Connecting Destination Ip&Port!")
			}
			logger.Errorln(er)
		}
		conn1.Write(socks4ReplyMessage(0x5B, 0, 0))
		return nil, er
	}
	n, er := conn1.Write(socks4ReplyMessage(0x5A, 0, 0))
	if er != nil || n < 8 {
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
