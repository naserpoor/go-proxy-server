package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	input := [2]byte{}
	server,_ := net.Listen("tcp", fmt.Sprintf("%v",os.Args[1]))
	fmt.Println("Server Started")

	for {
		conn,_ := server.Accept()
		fmt.Println("Connection Started")
		conn.Read(input[:])
		fmt.Println(input)
		if input[0] == 4 && input[1] == 1 {
			go socks4(conn)
		} else if input[0] == 5 && input[1] > 0 {
			go socks5(conn, input[1])
		} else {
			conn.Close()
		}
	}
}
