package main

import (
	"fmt"
	"net"
)

func director(conn1 net.Conn, conn2 net.Conn){
	input := makeInput(128)
	for{
		n,err := conn1.Read(input)
		if err != nil {
			fmt.Println("Error Socks Directing Read!:" + conn1.RemoteAddr().String())
			fmt.Println(err)
			conn2.Close()
			conn1.Close()
			return
		} else if n > 0 {
			_,err := conn2.Write(input[:n])
			if err != nil {
				fmt.Println("Error Socks Directing Write!:" + conn2.RemoteAddr().String())
				fmt.Println(err)
				conn1.Close()
				conn2.Close()
				return
			}
		}
	}
}

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
