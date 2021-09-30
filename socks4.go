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

	conn.Write(socks4_reply(0,90,0,0))
	go director(conn,conn2)
	go director(conn2,conn)
}

