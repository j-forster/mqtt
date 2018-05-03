# Golang MQTT-Server

This repo is a implementation of the [MQTT Protocol](http://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html)
for the [Go Programming Language](https://golang.org/).

## What is MQTT?

```
MQTT stands for MQ Telemetry Transport. It is a publish/subscribe, extremely
simple and lightweight messaging protocol, designed for constrained devices
and low-bandwidth, high-latency or unreliable networks. The design principles
are to minimise network bandwidth and device resource requirements whilst also
attempting to ensure reliability and some degree of assurance of delivery.
These principles also turn out to make the protocol ideal of the emerging
“machine-to-machine” (M2M) or “Internet of Things” world of connected devices,
and for mobile applications where bandwidth and battery power are at a premium.
```

→ See [MQTT FAQ](http://mqtt.org/faq)


## Install Go

```
Go is an open source programming language that makes it easy to build simple,
reliable, and efficient software.
```

You can download Go at [golang.org/dl](https://golang.org/dl/). There are
executables for Microsoft Windows, Apple macOS and Linux, as well as the Go
source code. There are also Golang releases for
[![Docker Logo](https://www.docker.com/favicon/favicon-16x16.png) Docker](https://hub.docker.com/_/golang/)!

## Run the MQTT-Server

First of all, grab this repository by running the following line. The `go get`
command will clone this repo into the `$GOPATH` directory.
```bash
go get github.com/j-forster/mqtt
```


Now build and install the server with:
```bash
go install github.com/j-forster/mqtt/server
```
This will compile the server code and move the executable to `$GOPATH/bin/`.

To start the server:
```bash
$GOPATH/bin/server
```

The default MQTT (TCP) port is `:1883`. You can now connect with any MQTT
client.