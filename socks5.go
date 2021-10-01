package main

import (
	"encoding/binary"
	"fmt"
	"net"
)

func socks5(conn net.Conn,n_auth byte) {
	input := make([]byte,n_auth)

	conn.Read(input)
	fmt.Println(input)

	var has_no_auth = false
	for _, auth_type := range input {
		if auth_type == 0 {
			has_no_auth = true
		}
	}
	if !has_no_auth {
		conn.Write([]byte{5,0xff})
		conn.Close()
		return
	} else {
		conn.Write([]byte{5,0})
	}

	input = make([]byte,10)
	conn.Read(input)
	fmt.Println(input)
	if input[0] != 5 || input[1] != 1 || input[2] != 0 || input[3] != 1 {
		conn.Close()
		return
	}


	var port uint16
	port = binary.BigEndian.Uint16(input[8:10])

	_,err := net.Dial("tcp",fmt.Sprintf("%s:%d",net.IPv4(input[4],input[5],input[6],input[7]).String(),port))
	if err != nil {
		fmt.Println(err)
		conn.Close()
		return
	}

	conn.Write([]byte{5,0,0,1,0,0,0,0,0,0})

}