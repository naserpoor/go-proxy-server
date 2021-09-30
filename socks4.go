package main

import (
	"encoding/binary"
	"fmt"
	"net"
)

func socks4_reply(vn byte, result byte, ip uint32, port uint16) []byte {
	var res = []byte{vn,result,0,0,0,0,0,0}
	binary.LittleEndian.PutUint32(res[2:6],ip)
	binary.LittleEndian.PutUint16(res[2:6],port)
	return res
}

func socks4(conn net.Conn) {
	input := [16]byte{}
	conn.Read(input[:])
	fmt.Println(input)

	var port uint16
	port = binary.BigEndian.Uint16(input[0:2])

	conn2,err := net.Dial("tcp",fmt.Sprintf("%s:%d",net.IPv4(input[2], input[3], input[4], input[5]).String(),port))
	if err != nil {
		fmt.Println(err)
		conn.Close()
		return
	}

	conn.Write(socks4_reply(0,90,0,0))
	go director(conn,conn2)
	go director(conn2,conn)
}

