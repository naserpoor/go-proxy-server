package main

import (
	"github.com/google/logger"
	"net"
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
