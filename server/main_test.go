package main

import (
	"bytes"
	"math/rand"
	"net"
	"testing"
)

func TestServer(t *testing.T) {

	CONNECT := []byte{
		// fixed header:
		0x10, // CONNECT
		0x1d, // remaining length (29 bytes)

		// variable header:
		0x00, 0x06, // protocol name length
		0x4d, 0x51, 0x49, 0x73, 0x64, 0x70, // protocol name "MQIsdp"
		0x03,       // mqtt protocol version 3
		0x01,       // connect flags (clean session: 1)
		0x00, 0x3c, // keep alive timer: 60s

		// payload
		0x00, 0x0f, // client id length
		0x48, 0x49, 0x4d, 0x51, 0x54, 0x54, 0x2d, 0x54, 0x65, 0x73, 0x74, // client id "HIMQTT-Test"
		0x00, 0x00, 0x00, 0x00} // id appendix (num_client as hex)

	conn, err := net.Dial("tcp", ":1883")
	if err != nil {
		t.Errorf("TCP conn error: %v", err)
		return
	}

	conn.Write(CONNECT)

	CONNACK := []byte{
		// fixed header:
		0x20, // CONNACK
		0x02, // remaining length (2 bytes)

		// variable header:
		0x00, 0x00} // return code 0 (Connection Accepted)

	buf := make([]byte, len(CONNACK))
	n, err := conn.Read(buf)
	if err != nil {
		t.Errorf("TCP conn read err: %v", err)
		return
	}
	if n != len(CONNACK) || bytes.Compare(CONNACK, buf) != 0 {
		t.Errorf("MQTT CONNACK failed:\nExpected: %v\nGot     : %v", CONNACK, buf)
		return
	}

	//////////////////////////////////////////////////////////////////////////////

	SUBSCRIBE := []byte{
		// fixed header:
		0x82, // SUBSCRIBE (qos 1)
		0x08, // remaining length (8 bytes)

		// variable header:
		0x12, 0x34, // message id

		0x00, 0x03, // topic name length
		0x61, 0x2f, 0x62, // topic "a/b"
		0x00} // qos 0

	conn.Write(SUBSCRIBE)

	SUBACK := []byte{
		// fixed header:
		0x90, // SUBACK
		0x03, // remaining length (3 bytes)

		// variable header:
		0x12, 0x34, // message id

		// payload:
		0x00} // granted qos (0)

	buf = make([]byte, len(SUBACK))
	n, err = conn.Read(buf)
	if err != nil {
		t.Errorf("TCP conn read err: %v", err)
		return
	}
	if n != len(SUBACK) || bytes.Compare(SUBACK, buf) != 0 {
		t.Errorf("MQTT SUBACK failed:\nExpected: %v\nGot     : %v", SUBACK, buf)
		return
	}

	//////////////////////////////////////////////////////////////////////////////

	PUBLISH_HEAD := []byte{
		// fixed header:
		0x30, // PUBLISH (qos 0)
		0x45, // remaining length (69 bytes)

		// variable header:
		0x00, 0x03, // topic name length
		0x61, 0x2f, 0x62} // topic "a/b"

	PUBLISH_BODY := make([]byte, 64)
	rand.Read(PUBLISH_BODY)

	conn.Write(PUBLISH_HEAD)
	conn.Write(PUBLISH_BODY)

	//////////////////////////////////////////////////////////////////////////////

	buf = make([]byte, 7)
	n, err = conn.Read(buf)
	if err != nil {
		t.Errorf("TCP conn read err: %v", err)
		return
	}
	if n != 7 || bytes.Compare(PUBLISH_HEAD, buf) != 0 {
		t.Errorf("MQTT PUBLISH HEAD recv failed:\nExpected: %v\nGot     : %v", PUBLISH_HEAD, buf)
		return
	}

	buf = make([]byte, 64)
	n, err = conn.Read(buf)
	if err != nil {
		t.Errorf("TCP conn read err: %v", err)
		return
	}
	if n != 64 || bytes.Compare(PUBLISH_BODY, buf) != 0 {
		t.Errorf("MQTT PUBLISH BODY recv failed:\nExpected: %v\nGot     : %v", PUBLISH_BODY, buf)
		return
	}

	//////////////////////////////////////////////////////////////////////////////

	DISCONNECT := []byte{
		// fixed header:
		0xe0, // DISCONNECT (qos 0)
		0x00} // remaining length (0 bytes)

	conn.Write(DISCONNECT)
	conn.Close()
}
