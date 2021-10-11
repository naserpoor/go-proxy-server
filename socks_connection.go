package main

import (
	"github.com/google/logger"
	"net"
	"reflect"
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
type SocksDirectingState struct {
	conn  net.Conn
	conn2 net.Conn
}

func (s SocksRecognizerState) ProcessData() (State, error) {
	input := make([]byte, 1)
	_, err := readFromConnection(s.conn, input, "Error Reading Socks Ver From Connection!")
	if err != nil {
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

func (s SocksDirectingState) ProcessData() (State, error) {
	chan1 := connToChannel(s.conn)
	chan2 := connToChannel(s.conn2)

	for {
		select {
		case input, ok := <-chan1:
			er := forwardReadDataToConnection(s.conn2, input, ok)
			if er != nil {
				return nil, er
			}
		case input, ok := <-chan2:
			er := forwardReadDataToConnection(s.conn, input, ok)
			if er != nil {
				return nil, er
			}
		}
	}
}


func forwardReadDataToConnection(conn net.Conn, input []byte, ok bool) error {
	if !ok {
		err := conn.Close()
		if err != nil {
			return err
		}
		return errorT{"Error Socks Channel Closed!"}
	}
	n, err := writeToConnection(conn, input, "Error Directing Socks Write!")
	if err != nil {
		return err
	}
	if *lpackets && *logging {
		logger.Infof("%v Bytes Written To %v\n", n, conn.RemoteAddr().String())
	}
	return nil
}

