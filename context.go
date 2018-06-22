package mqtt

import (
	"fmt"
	"io"
)

const (
	CONNECTING = 0
	CONNECTED  = 1
	CLOSING    = 3
	CLOSED     = 4
)

type Publisher interface {
	Publish(msg *Message)
}

type SubscriptionHandler interface {
	Subscribe(ctx *Context, topic string, qos byte) *Subscription
	Unsubscribe(subs *Subscription)
}

type Context struct {
	writer io.Writer
	closer io.Closer
	server *Server
	// publisher Publisher
	// subsHandler SubscriptionHandler

	//	server   *Server
	ClientID string

	state int

	mid int

	Will *Message

	messages map[int]*Message
	subs     map[string]*Subscription
	values   map[string]interface{}
}

func NewContext(w io.Writer, c io.Closer, server *Server) *Context {

	ctx := &Context{
		writer:   w,
		closer:   c,
		server:   server,
		messages: make(map[int]*Message),
		values:   make(map[string]interface{}),
		subs:     make(map[string]*Subscription)}

	return ctx
}

func (ctx *Context) Get(key string) interface{} {

	v, ok := ctx.values[key]
	if ok {
		return v
	}
	return nil
}

func (ctx *Context) Set(key string, value interface{}) {

	ctx.values[key] = value
}

func (ctx *Context) Alive() bool {

	return ctx.state != CLOSED
}

func (ctx *Context) Write(data []byte) (n int, err error) {
	n, err = ctx.writer.Write(data)
	return
}

func (ctx *Context) Close() error {

	if ctx.state != CLOSED {

		ctx.state = CLOSED

		for _, sub := range ctx.subs {
			//ctx.server
			ctx.server.Unsubscribe(sub)
		}

		ctx.subs = nil

		if ctx.closer != nil {
			ctx.closer.Close()
		}

		if ctx.server.handler != nil {
			ctx.server.handler.Disconnect(ctx)
		}
	}
	return nil
}

func (ctx *Context) Fail(err error) error {

	if ctx.Alive() {

		fmt.Println(err)
		ctx.Close()

		if ctx.Will != nil {
			ctx.server.Publish(ctx, ctx.Will)
		}
	}

	return err
}

func (ctx *Context) Failf(format string, a ...interface{}) error {
	return ctx.Fail(fmt.Errorf(format, a...))
}

// sowas wie Body() oder New() weil mal mit body und mal nur head benötigt wird..
func Head(b0 byte, length int, total int) ([]byte, []byte) {

	if length < 0x80 {
		buf := make([]byte, 2+total)
		buf[0] = b0
		buf[1] = byte(length)
		return buf, buf[2:]
	}

	if length < 0x8000 {
		buf := make([]byte, 3+total)
		buf[0] = b0
		buf[1] = byte(length & 127)
		buf[2] = byte(length >> 7)
		return buf, buf[3:]
	}

	if length < 0x800000 {
		buf := make([]byte, 4+total)
		buf[0] = b0
		buf[1] = byte(length & 127)
		buf[2] = byte((length >> 7) & 127)
		buf[3] = byte(length >> 14)
		return buf, buf[4:]
	}

	if length < 0x80000000 {
		buf := make([]byte, 5+total)
		buf[0] = b0
		buf[1] = byte(length & 127)
		buf[2] = byte((length >> 7) & 127)
		buf[3] = byte((length >> 14) & 127)
		buf[4] = byte(length >> 21)
		return buf, buf[5:]
	}

	return nil, nil
}

func (ctx *Context) ConnAck(code byte) {

	// buf, _ := WriteBegin(1)
	buf := make([]byte, 4)
	buf[0] = 0x20 // CONNACK
	buf[1] = 0x02 // remaining length: 2
	buf[3] = code

	ctx.Write(buf)
	if code != 0 {
		ctx.Close()
	}
}

func (ctx *Context) Subscribe(topic string, qos byte) byte {

	sub, ok := ctx.subs[topic]
	if !ok {
		//sub = new(Subscription)
		//sub.ctx = ctx
		//sub.qos = qos
		//ctx.server.Subscribe(topic, sub)
		sub = ctx.server.Subscribe(ctx, topic, qos)

		if sub != nil {

			ctx.subs[topic] = sub
		} else {

			// could not subscribe (the server is closing)
			ctx.Close()
			return 0
		}
	}

	//TODO it's not qos, but sub.qos
	// need to update the qos at the stored subscription
	// (the client may subscribe to an already subscribed topic)
	return qos // granted qos
}

func (ctx *Context) Publish(sub *Subscription, msg *Message) {

	// qos = Min(sub.qos, msg.qos)
	qos := sub.qos
	if msg.QoS < qos {
		qos = msg.QoS
	}

	switch qos {
	case 0:
		l := len(msg.Topic)
		head, vhead := Head(0x30|bool2byte(msg.retain), 2+l+len(msg.Buf), 2+l)
		vhead[0] = byte(l >> 8)
		vhead[1] = byte(l & 0xff)
		copy(vhead[2:], msg.Topic)
		ctx.Write(head)
		ctx.Write(msg.Buf)
	case 1, 2:
		l := len(msg.Topic)
		head, vhead := Head(0x30|(qos<<1)|bool2byte(msg.retain), 2+l+2+len(msg.Buf), 2+l+2)
		vhead[0] = byte(l >> 8)
		vhead[1] = byte(l & 0xff)
		copy(vhead[2:], msg.Topic)
		ctx.mid++
		vhead[2+l] = byte(ctx.mid >> 8)
		vhead[2+l+1] = byte(ctx.mid & 0xff)
		ctx.Write(head)
		ctx.Write(msg.Buf)

		//TODO store message and retry if timeout
	}
}

func (ctx *Context) Unsubscribe(topic string) {

	sub, ok := ctx.subs[topic]
	if ok {
		ctx.server.Unsubscribe(sub)
	}
}

func (ctx *Context) PingResp() {
	// buf, _ := WriteBegin(0)
	buf := make([]byte, 2)
	buf[0] = 0xD0 // PINGRESP
	buf[1] = 0x00 // remaining length: 0
	ctx.Write(buf)
}

///////////////////////////////////////////////////////////////////////////////

// what is wrong with golang to not support b := byte(a bool) ?!
func bool2byte(a bool) byte {
	if a {
		return 1
	}
	return 0
}
