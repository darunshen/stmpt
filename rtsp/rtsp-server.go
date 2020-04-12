package rtsp

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/darunshen/stmpts/protocolinterface"
	"github.com/teris-io/shortid"
)

var (
	//ReadBufferSize bio&tcp&udp read buffer size
	ReadBufferSize int
	//WriteBufferSize bio&tcp&udp write buffer size
	WriteBufferSize int
	//PushChannelBufferSize pusher channel buffer size
	PushChannelBufferSize int
	//PullChannelBufferSize puller channel buffer size
	PullChannelBufferSize int
)

// Server rtsp server
type Server struct {
	protocolinterface.BasicNet
	PusherPullersSessionMap      map[string]*PusherPullersSession
	PusherPullersSessionMapMutex sync.Mutex
}

// StartSession start a session with rtsp client
func (server *Server) StartSession(conn *net.TCPConn) error {
	fmt.Println(
		"start session from ",
		conn.LocalAddr().String(),
		"to", conn.RemoteAddr().String())

	newSession := new(NetSession)
	newSession.Conn = conn
	host := newSession.Conn.RemoteAddr().String()
	rip := host[:strings.LastIndex(host, ":")]
	newSession.RemoteIP = &rip
	newSession.PusherPullersSessionMap = server.PusherPullersSessionMap
	newSession.PusherPullersSessionMapMutex = &server.PusherPullersSessionMapMutex
	newSession.Bufio =
		bufio.NewReadWriter(
			bufio.NewReaderSize(conn, ReadBufferSize),
			bufio.NewWriterSize(conn, WriteBufferSize))
	newSession.ID = shortid.MustGenerate()
	for {
		if pkg, err := newSession.ReadPackage(); err == nil {
			if err = newSession.ProcessPackage(pkg); err != nil {
				fmt.Printf("ProcessPackage error:%v\n", err)
				break
			}
			if err = newSession.WritePackage(pkg); err != nil {
				fmt.Printf("WritePackage error:%v\n", err)
				break
			}
		} else {
			fmt.Printf("ReadPackage error:%v\n", err)
			break
		}
	}
	if err := newSession.CloseSession(); err != nil {
		return fmt.Errorf("CloseSession error:%v", err)
	}
	return fmt.Errorf("StartSession error")
}

/*Start start a rtsp server
bufferReadSize: bio&tcp read buffer size
bufferWriteSize: bio&tcp write buffer size
*/
func (server *Server) Start(address string,
	bufferReadSize int,
	bufferWriteSize int,
	pushChannelBufferSize int,
	pullChannelBufferSize int) error {
	ReadBufferSize = bufferReadSize
	WriteBufferSize = bufferWriteSize
	PushChannelBufferSize = pushChannelBufferSize
	PullChannelBufferSize = pullChannelBufferSize
	server.PusherPullersSessionMap = make(map[string]*PusherPullersSession)
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return fmt.Errorf("address resolving failed : %v", err)
	}
	server.Host, _, _ = net.SplitHostPort(address)
	server.Port = addr.Port
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen tcp failed : %v", err)
	}
	fmt.Println("Start listening at ", address)
	server.TCPListener = listener

	server.IfStop = false
	for !server.IfStop {
		conn, err := server.TCPListener.AcceptTCP()
		if err != nil {
			fmt.Println("AcceptTCP failed : ", err)
			continue
		}
		if err := conn.SetReadBuffer(ReadBufferSize); err != nil {
			return fmt.Errorf("SetReadBuffer error, %v", err)
		}
		if err := conn.SetWriteBuffer(WriteBufferSize); err != nil {
			return fmt.Errorf("SetWriteBuffer error, %v", err)
		}
		go server.StartSession(conn)
	}
	return nil
}
