package main

import (
	"encoding/binary"
	"fmt"
	"net"
)

func socks4(conn net.Conn) {
	inputBuffer := [16]byte{}
	conn.Read(inputBuffer[:])

	fmt.Println(inputBuffer)
	var port uint16
	port = binary.BigEndian.Uint16(inputBuffer[0:2])

	conn2,err := net.Dial("tcp",fmt.Sprintf("%s:%d",net.IPv4(inputBuffer[2],inputBuffer[3],inputBuffer[4],inputBuffer[5]).String(),port))
	if err != nil {
		fmt.Println(err)
		conn.Close()
		return
	}

	conn.Write([]byte{0,90,0,0,0,0,0,0})
	go director(conn,conn2)
	go director(conn2,conn)
}

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