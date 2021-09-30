package main

import (
	"fmt"
	"net"
)

func director(conn1 net.Conn, conn2 net.Conn){
	input := [128]byte{}
	for{
		n,err := conn1.Read(input[:])
		if err != nil {
			fmt.Println(err)
			conn2.Close()
			conn1.Close()
			return
		} else if n > 0 {
			_,err := conn2.Write(input[:n])
			if err != nil {
				fmt.Println(err)
				conn1.Close()
				conn2.Close()
				return
			}
		}
	}
}
