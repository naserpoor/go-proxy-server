package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	server,_ := net.Listen("tcp", fmt.Sprintf("%v",os.Args[1]))
	fmt.Println("Server Started")

	for {
		conn,_ := server.Accept()
		go func() {
			input := make([]byte,2)
			fmt.Println("Connection Started")
			_,err := conn.Read(input[:])
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(input)
			if input[0] == 4 && input[1] == 1 {
				go socks4(conn)
			} else if input[0] == 5 && input[1] > 0 {
				go socks5(conn, input[1])
			} else {
				conn.Close()
			}
		}()
	}
}
