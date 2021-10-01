package main

import (
	"fmt"
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
			input := makeInput(128)
			n ,er := conn.Read(input)
			if er != nil {
				fmt.Println("Error Read From:" + conn.RemoteAddr().String())
				fmt.Println(er)
				close(channel)
				return
			}
			if n > 0 {
				channel <- input[:n]
			}
		}

	}()
	return channel
}
