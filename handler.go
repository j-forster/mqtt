package mqtt

type Handler interface {
	Connect(ctx *Context, username, password string) error
	Disconnect(ctx *Context)
	Publish(ctx *Context, msg *Message) error
	Subscribe(ctx *Context, topic string, qos byte) error
}
