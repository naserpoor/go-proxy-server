package main

import (
	goSocks5 "github.com/armon/go-socks5"
	"gopkg.in/elazarl/goproxy.v1"
	"log"
	"net/http"
)

func main() {
	//server,_ := net.Listen("tcp", fmt.Sprintf("%v",os.Args[1]))
	//fmt.Println("Server Started")

	//http.HandleFunc("/",http_proxy)
	//http.ListenAndServe(":12012",nil)
	go func() {
		proxy := goproxy.NewProxyHttpServer()
		log.Fatal(http.ListenAndServe(":12012", proxy))
	}()

	go func() {
		conf := &goSocks5.Config{}
		server, err := goSocks5.New(conf)
		if err != nil {
			panic(err)
		}

		// Create SOCKS5 proxy on localhost port 8000
		if err := server.ListenAndServe("tcp", ":12013"); err != nil {
			panic(err)
		}
	}()

	//for {
	//	conn,_ := server.Accept()
	//	go func() {
	//		input := make([]byte,2)
	//		fmt.Println("Connection Started")
	//		_,err := conn.Read(input[:])
	//		if err != nil {
	//			fmt.Println(err)
	//			return
	//		}
	//		fmt.Println(input)
	//		if input[0] == 4 && input[1] == 1 {
	//			go socks4(conn)
	//		} else if input[0] == 5 && input[1] > 0 {
	//			go socks5(conn, input[1])
	//		} else {
	//			conn.Close()
	//		}
	//	}()
	//}
}
