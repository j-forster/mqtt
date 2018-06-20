package main

import (
	"log"

	"github.com/j-forster/mqtt"
	// "net/http"
	//  _ "net/http/pprof"
)

type SimpleHandler struct{}

func (h *SimpleHandler) Connect(ctx *mqtt.Context, username, password string) error {

	log.Printf("%v Connected: '%v' '%v'", ctx.ClientID, username, password)
	return nil // no error == accept everyone
}

func (h *SimpleHandler) Disconnect(ctx *mqtt.Context) {

	log.Printf("%v Disconnected", ctx.ClientID)
}

func (h *SimpleHandler) Publish(ctx *mqtt.Context, msg *mqtt.Message) error {

	log.Printf("%v Publish: '%v' [%v]", ctx.ClientID, msg.Topic, len(msg.Buf))
	return nil // no error == no message filter
}

func (h *SimpleHandler) Subscribe(ctx *mqtt.Context, topic string, qos byte) error {

	log.Printf("%v Subscribe: '%v'", ctx.ClientID, topic)
	return nil // no error == no subscribe filter
}

func main() {

	var handler SimpleHandler
	log.Println("Up and running: Port 1883")
	mqtt.ListenAndServe(":1883", &handler)
}
