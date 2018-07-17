package mqtt

import (
	"errors"
	"io"
	"log"
)

// errors
var (
	InclompleteHeader       = errors.New("incomplete header")
	MaxMessageLength        = errors.New("message length exceeds server maximum")
	MessageLengthInvalid    = errors.New("message length exceeds maximum")
	IncompleteMessage       = errors.New("incomplete message")
	UnknownMessageType      = errors.New("unknown mqtt message type")
	ReservedMessageType     = errors.New("reserved message type")
	ConnectMsgLacksProtocol = errors.New("connect message has no protocol field")
	ConnectProtocolUnexp    = errors.New("connect message protocol is not 'MQIsdp'")
	TooLongClientID         = errors.New("connect client id is too long")
	UnknownMessageID        = errors.New("unknown message id")
)

const maxMessageLength = 1024 * 1024 * 6

// CONNACK return codes
const (
	ACCEPTED            = 0
	UNACCEPTABLE_PROTOV = 1
	IDENTIFIER_REJ      = 2
	SERVER_UNAVAIL      = 3
	BAD_USER_OR_PASS    = 4
	NOT_AUTHORIZED      = 5
)

// message types
const (
	CONNECT     = 1
	CONNACK     = 2
	PUBLISH     = 3
	PUBACK      = 4
	PUBREC      = 5
	PUBREL      = 6
	PUBCOMP     = 7
	SUBSCRIBE   = 8
	SUBACK      = 9
	UNSUBSCRIBE = 10
	UNSUBACK    = 11
	PINGREQ     = 12
	PINGRESP    = 13
	DISCONNECT  = 14
)

// string representation of message types
var messageType = [...]string{"reserved", "CONNECT", "CONNACK", "PUBLISH",
	"PUBACK", "PUBREC", "PUBREL", "PUBCOMP", "SUBSCRIBE", "SUBACK", "UNSUBSCRIBE",
	"UNSUBACK", "PINGREQ", "PINGRESP", "DISCONNECT"}

///////////////////////////////////////////////////////////////////////////////

type Message struct {
	Topic  string
	Buf    []byte
	QoS    byte
	retain bool
}

///////////////////////////////////////////////////////////////////////////////

func readString(buf []byte) (int, string) {
	length, b := readBytes(buf)
	return length, string(b)
}

func readBytes(buf []byte) (int, []byte) {

	if len(buf) < 2 {
		return 0, nil
	}
	length := (int(buf[0])<<8 + int(buf[1])) + 2
	if len(buf) < length {
		return 0, nil
	}
	return length, buf[2:length]
}

///////////////////////////////////////////////////////////////////////////////

type FixedHeader struct {
	mtype  byte
	dup    bool
	qos    byte
	retain bool
	length int
}

func (fh *FixedHeader) Read(reader io.Reader) error {

	var headBuf [1]byte
	n, err := reader.Read(headBuf[:])
	if err != nil {
		return err // read error
	}
	if n == 0 {
		return io.EOF // connection closed
	}

	fh.mtype = byte(headBuf[0] >> 4)
	fh.dup = bool(headBuf[0]&0x8 != 0)
	fh.qos = byte((headBuf[0] & 0x6) >> 1)
	fh.retain = bool(headBuf[0]&0x1 != 0)

	if fh.mtype == 0 || fh.mtype == 15 {
		return ReservedMessageType // reserved type
	}

	var multiplier int = 1
	var length int

	for {
		n, err = reader.Read(headBuf[:])
		if err != nil {
			return err // read error
		}
		if n == 0 {
			return InclompleteHeader // connection closed in header
		}

		length += int(headBuf[0]&127) * multiplier

		if length > maxMessageLength {
			return MaxMessageLength // server maximum message size exceeded
		}

		if headBuf[0]&128 == 0 {
			break
		}

		if multiplier > 0x4000 {
			return MessageLengthInvalid // mqtt maximum message size exceeded
		}

		multiplier *= 128
	}

	fh.length = length
	return nil
}

///////////////////////////////////////////////////////////////////////////////

// read from a reader (input stream) a new mqtt message
func (ctx *Context) Read(reader io.Reader) {

	var fh FixedHeader
	if err := fh.Read(reader); err != nil {
		ctx.Fail(err)
		return
	}

	buf := make([]byte, fh.length)

	_, err := io.ReadFull(reader, buf)
	if err != nil {
		ctx.Fail(IncompleteMessage)
		return
	}

	// log.Printf("Message: %s (length:%d qos:%d dup:%t retain:%t)",
	//   messageType[fh.mtype],
	//   fh.length,
	//   fh.qos,
	//   fh.dup,
	//   fh.retain)

	switch fh.mtype {
	case CONNECT:
		ctx.ReadConnectMessage(reader, &fh, buf)
	case SUBSCRIBE:
		ctx.ReadSubscribeMessage(reader, &fh, buf)
	case PUBLISH:
		ctx.ReadPublishMessage(reader, &fh, buf)
	case PUBREL:
		ctx.ReadPubrelMessage(reader, &fh, buf)
	case PUBREC:
		ctx.ReadPubrecMessage(reader, &fh, buf)
	case PUBCOMP:
		ctx.ReadPubcompMessage(reader, &fh, buf)
	case PINGREQ:
		ctx.PingResp()
	case DISCONNECT:
		ctx.Close()
	}
}

///////////////////////////////////////////////////////////////////////////////

// parse CONNECT messages
func (ctx *Context) ReadConnectMessage(reader io.Reader, fh *FixedHeader, buf []byte) {
	l, protocol := readString(buf)
	if l == 0 {
		ctx.Fail(ConnectMsgLacksProtocol)
		return
	}
	if protocol != "MQIsdp" {
		ctx.Failf("unsupported protocol '%.12s'", protocol)
		return
	}
	buf = buf[l:]

	//

	if len(buf) < 1 {
		ctx.Fail(IncompleteMessage)
		return
	}
	version := buf[0]
	if version != 0x03 {
		ctx.ConnAck(UNACCEPTABLE_PROTOV)
		return
	}
	buf = buf[1:]

	//

	if len(buf) < 1 {
		ctx.Fail(IncompleteMessage)
		return
	}
	connFlags := buf[0]
	// log.Printf("Connection Flags: %d", connFlags)

	// cleanSession := connFlags&0x02 != 0
	// if cleanSession {
	// 	log.Println("Clean Session: true")
	// }
	willFlag := connFlags&0x04 != 0
	willQoS := connFlags & 0x18 >> 3
	willRetain := connFlags&0x20 != 0
	passwordFlag := connFlags&0x40 != 0
	usernameFlag := connFlags&0x80 != 0

	buf = buf[1:]

	//

	if len(buf) < 2 {
		ctx.Fail(IncompleteMessage)
		return
	}
	// keepAliveTimer := int(buf[0])<<8 + int(buf[1])
	// log.Printf("KeepAlive Timer: %d", keepAliveTimer)
	// TODO set SetDeadline() to conn
	buf = buf[2:]

	//

	l, ctx.ClientID = readString(buf)
	if l == 0 {
		ctx.Fail(IncompleteMessage)
		return
	}
	if l > 128 {
		// should be max 23, but some client implementations ignore this
		// so we increase the size to 128
		ctx.ConnAck(IDENTIFIER_REJ)
		return
	}
	buf = buf[l:]

	//

	if willFlag {

		var will Message

		will.retain = willRetain
		will.QoS = willQoS

		l, will.Topic = readString(buf)
		if l == 0 {
			ctx.Fail(IncompleteMessage)
			return
		}
		buf = buf[l:]

		l, will.Buf = readBytes(buf)
		if l == 0 {
			ctx.Fail(IncompleteMessage)
			return
		}

		log.Printf("Will: topic:%q qos:%d %q\n", will.Topic, will.QoS, will.Buf)

		ctx.Will = &will
		buf = buf[l:]
	}

	//

	var username, password string

	if usernameFlag {

		l, username = readString(buf)
		if l == 0 {
			ctx.Fail(IncompleteMessage)
			return
		}
		buf = buf[l:]

		if passwordFlag {

			l, password = readString(buf)
			if l != 0 {
				buf = buf[l:]
			}
		}
	}

	if ctx.server.handler != nil && ctx.server.handler.Connect(ctx, username, password) == nil {

		ctx.ConnAck(ACCEPTED)
	} else {

		if !usernameFlag {
			ctx.ConnAck(NOT_AUTHORIZED)
		} else {
			ctx.ConnAck(BAD_USER_OR_PASS)
		}
	}
}

///////////////////////////////////////////////////////////////////////////////

// parse a SUBSCRIBE message and send SUBACK
func (ctx *Context) ReadSubscribeMessage(reader io.Reader, fh *FixedHeader, buf []byte) {

	if len(buf) < 2 {
		ctx.Fail(IncompleteMessage)
		return
	}
	mid := int(buf[0])<<8 + int(buf[1])
	buf = buf[2:]
	var s int
	for i, l := 0, len(buf); i != l; s++ {

		i += (int(buf[i]) << 8) + int(buf[i+1]) + 2 + 1
		if i > l {
			ctx.Fail(IncompleteMessage)
			return
		}
	}

	l := 2 + s
	head, body := Head(0x90, l, l) // SUBACK
	body[0] = byte(mid >> 8)       // mid MSB
	body[1] = byte(mid & 0xff)     // mid LSB
	s = 2

	for len(buf) != 0 {
		l, topic := readString(buf)
		qos := buf[l] & 0x03
		buf = buf[l+1:]

		// grantedQos
		body[s] = ctx.Subscribe(topic, qos)
		s++
	}

	ctx.Write(head)
}

///////////////////////////////////////////////////////////////////////////////

// parse a PUBLISH message and tell the server about it
func (ctx *Context) ReadPublishMessage(reader io.Reader, fh *FixedHeader, buf []byte) {

	if len(buf) < 2 {
		ctx.Fail(IncompleteMessage)
		return
	}
	l, topic := readString(buf)
	if l == 0 {
		ctx.Fail(IncompleteMessage)
		return
	}
	buf = buf[l:]

	if fh.qos == 0 { // QoS 0

		ctx.server.Publish(ctx, &Message{topic, buf, 0, fh.retain})

	} else { // QoS 1 or 2

		if len(buf) < 2 {
			ctx.Fail(IncompleteMessage)
			return
		}
		mid := int(buf[0])<<8 + int(buf[1])
		buf = buf[2:]

		msg := &Message{topic, buf, fh.qos, fh.retain}

		if fh.qos == 1 {

			ctx.server.Publish(ctx, msg)

			// send PUBACK message
			buf := make([]byte, 4)
			buf[0] = 0x40 // PUBACK
			buf[1] = 0x02 // remaining length: 2
			buf[2] = byte(mid >> 8)
			buf[3] = byte(mid & 0xff)
			ctx.Write(buf)
		} else {

			ctx.messages[mid] = msg // store

			// send PUBREC message
			buf := make([]byte, 4)
			buf[0] = 0x50 // PUBREC
			buf[1] = 0x02 // remaining length: 2
			buf[2] = byte(mid >> 8)
			buf[3] = byte(mid & 0xff)
			ctx.Write(buf)
		}
	}
}

///////////////////////////////////////////////////////////////////////////////

// parse a PUBREL message (a response to a PUBREC at QoS 2)
// the message has alredy been stored at the previous PUBREC message
func (ctx *Context) ReadPubrelMessage(reader io.Reader, fh *FixedHeader, buf []byte) {

	if len(buf) < 2 {
		ctx.Fail(IncompleteMessage)
		return
	}
	mid := int(buf[0])<<8 + int(buf[1])

	msg, ok := ctx.messages[mid]
	if !ok {
		ctx.Fail(UnknownMessageID)
		return
	}

	ctx.server.Publish(ctx, msg)
	delete(ctx.messages, mid)

	// send PUBREC message
	buf = make([]byte, 4)
	buf[0] = 0x70 // PUBCOMP
	buf[1] = 0x02 // remaining length: 2
	buf[2] = byte(mid >> 8)
	buf[3] = byte(mid & 0xff)
	ctx.Write(buf)
}

///////////////////////////////////////////////////////////////////////////////

// parse a PUBREC message
// (a response to a publish from this server to a client on qos 2)
func (ctx *Context) ReadPubrecMessage(reader io.Reader, fh *FixedHeader, buf []byte) {

	if len(buf) < 2 {
		ctx.Fail(IncompleteMessage)
		return
	}
	mid := int(buf[0])<<8 + int(buf[1])

	// send PUBREL message
	buf = make([]byte, 4)
	buf[0] = 0x62 // PUBREL at qos 1
	buf[1] = 0x02 // remaining length: 2
	buf[2] = byte(mid >> 8)
	buf[3] = byte(mid & 0xff)
	ctx.Write(buf)
}

///////////////////////////////////////////////////////////////////////////////

// parse a PUBCOMP message
// (a response to a PUBREL from a client to this server)
func (ctx *Context) ReadPubcompMessage(reader io.Reader, fh *FixedHeader, buf []byte) {

	if len(buf) < 2 {
		ctx.Fail(IncompleteMessage)
		return
	}
	// mid := int(buf[0])<<8 + int(buf[1])
}

//////////////////////////////////////////////////////////////////////////////
