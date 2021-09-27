package main

import "net"

func main() {
	input := [2]byte{}
	server,_ := net.Listen("tcp","localhost:1080")

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
