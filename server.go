package mqtt

import (
	"net"
	"log"
	"strings"
	"io"
	"github.com/j-forster/mqtt/tools"
)

type SubscriptionRequest struct {
	subs *Subscription
	topic string
}

const (
	CREATE = 1
	REMOVE = 2
)

type SubscriptionChange struct {
	action int
	subs *Subscription
	topic string
}

type Server struct {

	//subsReq chan SubscriptionRequest
	//unsubs chan *Subscription
	state int
	closer io.Closer
	sigclose chan (struct{})
	subs chan SubscriptionChange
	pub chan *Message
	topics *Topic
}


func NewServer(closer io.Closer) (*Server) {

	svr := new(Server)
	//svr.subsReq = make(chan SubscriptionRequest)
	//svr.unsubs = make(chan *Subscription)
	svr.closer = closer;
	svr.sigclose = make(chan struct{})
	svr.subs = make(chan SubscriptionChange)
	svr.pub = make(chan *Message)
	svr.topics = NewTopic(nil, "")
	return svr
}

func (svr *Server) Alive() bool {

	return svr.state != CLOSING && svr.state != CLOSED
}

func (svr *Server) Publish(msg *Message) {

	if ! svr.Alive() {
		return
	}

	svr.pub <- msg
}

func (svr *Server) Subscribe(ctx *Context, topic string, qos byte) (*Subscription){

	if ! svr.Alive() {
		return nil
	}

	subs := NewSubscription(ctx, qos)
	svr.subs <- SubscriptionChange{CREATE, subs, topic}
	return subs
}


func (svr *Server) Unsubscribe(subs *Subscription) {

	if ! svr.Alive() {
		return
	}

	svr.subs <- SubscriptionChange{REMOVE, subs, ""}
}

func (svr *Server) Run() {

	RUN:
	for {
		select {
		case <- svr.sigclose:

			close(svr.subs)
			close(svr.pub)

			SYSALL := []string{"$SYS", "all"}
			subs := svr.topics.Find(SYSALL)
			for ;subs != nil; subs = subs.next {

				subs.ctx.Close()
			}

			svr.state = CLOSED
			break RUN

		case evt := <-svr.subs:

			switch evt.action {
			case CREATE:
				svr.topics.Subscribe(strings.Split(evt.topic, "/"), evt.subs)

			case REMOVE:
				evt.subs.Unsubscribe()
			}

			// log.Println("Topics:", svr.topics)

		case msg := <-svr.pub:

			if msg.topic == "$SYS/close" {
				svr.Close()

			} else {

				log.Printf("Publish: %s %q", msg.topic, string(msg.buf))
				svr.topics.Publish(strings.Split(msg.topic, "/"), msg)
			}
		}
	}
}


func (svr *Server) Close() {

	if svr.Alive() {

		close(svr.sigclose)
		svr.state = CLOSING
		if svr.closer != nil {
			svr.closer.Close()
		}
	}
}


func Join(conn net.Conn, p Publisher, s SubscriptionHandler) {

	uconn := tools.Unblock(conn)

	ctx := NewContext(uconn, uconn, p, s)
	defer ctx.Close()

	ctx.Subscribe("$SYS/all", 0)

	for ctx.Alive() {
		ctx.Read(conn)
	}
}
