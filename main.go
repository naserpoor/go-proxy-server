package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	input := [2]byte{}
	server,_ := net.Listen("tcp", fmt.Sprintf("%v",os.Args[1]))

	for {
		conn,_ := server.Accept()
		conn.Read(input[:])
		if input[0] == 4 && input[1] == 1 {
			go socks4(conn)
		} else {
			conn.Close()
		}
	}
}
