package main

import (
	"log"
	"net"

	"github.com/j-forster/mqtt"
	// "net/http"
	//  _ "net/http/pprof"
	"time"
)

func Server(address string) *mqtt.Server {

	tcp, err := net.Listen("tcp", address)
	if err != nil {
		log.Panicln(err)
	}

	server := mqtt.NewServer(tcp)

	log.Println("Up and running:", tcp.Addr())

	//go func() {
	//	log.Println(http.ListenAndServe("localhost:6060", nil))
	//}()

	go func() {

		for {

			conn, err := tcp.Accept()
			if err == nil {

				log.Println("Accepted", conn.RemoteAddr())
				go mqtt.Join(conn, server, server)
			} else {

				log.Fatal(err)
				break
			}
		}

		log.Println("Bye!")
	}()

	return server
}

func main() {

	srv := Server(":1883")
	srv.Run()
	time.Sleep(time.Minute)
}
