package protocolinterface

import (
	"bufio"
	"net"
)

// BasicNet support basic operation for net interface
type BasicNet struct {
	NetServer
	Host        string
	Port        int
	TCPListener *net.TCPListener
	IfStop      bool
}

// BasicNetSession support basic operation
// for session within server and client
type BasicNetSession struct {
	NetSession
	Conn     *net.TCPConn // connection from remote end point
	Bufio    *bufio.ReadWriter
	Extra    *interface{}
	ID       string  // session 's unique id
	RemoteIP *string // remote end point's ip of Conn
}
