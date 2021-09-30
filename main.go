package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
	input := [2]byte{}
	server,_ := net.Listen("tcp", fmt.Sprintf("%v",os.Args[1]))

	for {
		conn,_ := server.Accept()
		conn.SetReadDeadline(time.Now().Add(time.Second*5))
		conn.Read(input[:])
		conn.SetWriteDeadline(time.Now().Add(time.Second*5))
		conn.Write([]byte{12,11,10})
		conn.Close()
	}
}
