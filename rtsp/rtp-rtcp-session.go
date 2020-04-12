package rtsp

import (
	"fmt"
	"net"
	"regexp"
	"time"
)

//MediaType the media type of this session
type MediaType int

const (
	//MediaVideo stand for video
	MediaVideo MediaType = 0
	//MediaAudio stand for audio
	MediaAudio MediaType = 1
)

//ClientType rtsp client type(remote end point of connection)func (session *RtpRtcpSession)
type ClientType int

const (
	//PusherClient the rtsp client is pusher
	PusherClient ClientType = 0
	//PullerClient the rtsp client is puller
	PullerClient ClientType = 1
)

//RtpRtcpSession a pair of rtp-rtcp sessions
type RtpRtcpSession struct {
	RtspSessionID       string               // identification of rtsp session
	RtpUDPConnToPuller  *net.UDPConn         // rtp udp connection to puller
	RtcpUDPConnToPuller *net.UDPConn         // rtcp udp connection to puller
	RtpUDPConnToPusher  *net.UDPConn         // rtp udp connection to pusher
	RtcpUDPConnToPusher *net.UDPConn         // rtcp udp connection to pusher
	RtpServerPort       *string              // rtp Server port in udp session
	RtcpServerPort      *string              // rtcp Server port in udp session
	SessionMediaType    MediaType            // this session's media type
	SessionClientType   ClientType           // this session's client type(connected to pusher or puller)
	RtpPackageChannel   chan *RtpRtcpPackage // rtp packages for puller
	RtcpPackageChannel  chan *RtpRtcpPackage // rtcp packages for puller
	IfStop              bool                 // if stop is true,then stop go routines created by this session
	IfPause             bool                 // if pause transfer
}

//PackageType package type
type PackageType int

const (
	//RtpPackage stand for rtp
	RtpPackage PackageType = 0
	//RtcpPackage stand for rtcp
	RtcpPackage PackageType = 1
)

//RtpRtcpPackage store rtp/rtcp package content
type RtpRtcpPackage []byte

// type RtpRtcpPackage struct {
// 	DataType PackageType
// 	Data     *[]byte
// }

//PullerClientInfo the puller 's info as input to create rtp/rtcp
type PullerClientInfo struct {
	RtpRemotePort, RtcpRemotePort, IPRemote *string
}

//StartRtpRtcpSession Start a pair of rtp-rtcp sessions
func (session *RtpRtcpSession) StartRtpRtcpSession(
	clientType ClientType, mediaType MediaType,
	pullerClientInfo *PullerClientInfo, rtspSessionID string) error {
	var (
		err error
	)
	if rtspSessionID == "" {
		return fmt.Errorf("rtsp session id is empty")
	}
	session.RtspSessionID = rtspSessionID
	switch clientType {
	case PusherClient:
		session.RtpUDPConnToPusher, session.RtpServerPort, err =
			session.startUDPServer()
		if err != nil {
			return fmt.Errorf("startUDPServer failed : %v", err)
		}
		session.RtcpUDPConnToPusher, session.RtcpServerPort, err =
			session.startUDPServer()
		if err != nil {
			return fmt.Errorf("startUDPServer failed : %v", err)
		}
	case PullerClient:
		if pullerClientInfo == nil {
			return fmt.Errorf("StartRtpRtcpSession :pullerClientInfo is nil")
		}
		session.RtpUDPConnToPuller, err = session.startUDPClient(
			pullerClientInfo.IPRemote, pullerClientInfo.RtpRemotePort)
		if err != nil {
			return fmt.Errorf("startUDPClient failed : %v", err)
		}
		session.RtcpUDPConnToPuller, err = session.startUDPClient(
			pullerClientInfo.IPRemote, pullerClientInfo.RtcpRemotePort)
		if err != nil {
			return fmt.Errorf("startUDPClient failed : %v", err)
		}
		session.RtpPackageChannel = make(chan *RtpRtcpPackage, PullChannelBufferSize)
		session.RtcpPackageChannel = make(chan *RtpRtcpPackage, PullChannelBufferSize)
	default:
		return fmt.Errorf("clientType error,not support")
	}
	session.SessionMediaType = mediaType
	session.SessionClientType = clientType
	return nil
}

//startUDPServer start udp server for rtp/rtcp,
func (session *RtpRtcpSession) startUDPServer() (*net.UDPConn, *string, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		return nil, nil, err
	}
	udpConnection, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, nil, err
	}
	if err := udpConnection.SetReadBuffer(ReadBufferSize); err != nil {
		return nil, nil, err
	}
	if err := udpConnection.SetWriteBuffer(WriteBufferSize); err != nil {
		return nil, nil, err
	}
	port := regexp.MustCompile(":(\\d+)").
		FindStringSubmatch(udpConnection.LocalAddr().String())
	if port != nil {
		return udpConnection, &port[1], nil
	}
	return nil, nil, fmt.Errorf("not find udp port number in %v",
		udpConnection.LocalAddr().String())

}

//startUDPClient start udp connection to puller
func (session *RtpRtcpSession) startUDPClient(ip, port *string) (*net.UDPConn, error) {
	host := *ip + ":" + *port
	udpAddr, err := net.ResolveUDPAddr("udp", host)
	if err != nil {
		return nil, err
	}
	udpConnection, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, err
	}
	if err := udpConnection.SetReadBuffer(ReadBufferSize); err != nil {
		return nil, err
	}
	if err := udpConnection.SetWriteBuffer(WriteBufferSize); err != nil {
		return nil, err
	}
	return udpConnection, nil
}

//BeginTransfer begin recieving packages from pusher,then push into channel
func (session *RtpRtcpSession) BeginTransfer(rtpChan, rtcpChan chan RtpRtcpPackage) error {
	if session.IfPause {
		session.IfPause = false
		return nil
	}
	if session.SessionClientType == PusherClient &&
		(rtpChan == nil || rtcpChan == nil) {
		return fmt.Errorf(
			"BeginTransfer failed,PusherClient input channel arg have nil")
	}
	if session.SessionClientType == PullerClient &&
		!(rtpChan == nil && rtcpChan == nil) {
		return fmt.Errorf(
			"BeginTransfer failed,PullerClient input channel arg not all nil")
	}
	var mediaName string
	if session.SessionMediaType == MediaVideo {
		mediaName = "video"
	} else if session.SessionMediaType == MediaAudio {
		mediaName = "audio"
	}
	if session.SessionClientType == PusherClient {
		go func() {
			var num int = 0
			data := make([]byte, ReadBufferSize)
			for !session.IfStop {
				for session.IfPause {
					time.Sleep(time.Duration(10) * time.Millisecond)
				}
				if number, _, err := session.RtpUDPConnToPusher.ReadFromUDP(data); err == nil {
					buf := make([]byte, number)
					copy(buf, data)
					rtpChan <- buf
					num++
					if num%200 == 0 {
						fmt.Println(mediaName, "rtp pusher recieved data number =", num)
					}
				} else {
					fmt.Printf("error occured when read from pusher = %v\n", err)
					return
				}
			}
		}()
		go func() {
			var num int = 0
			data := make([]byte, ReadBufferSize)
			for !session.IfStop {
				for session.IfPause {
					time.Sleep(time.Duration(10) * time.Millisecond)
				}
				if number, _, err := session.RtcpUDPConnToPusher.ReadFromUDP(data); err == nil {
					buf := make([]byte, number)
					copy(buf, data)
					rtcpChan <- buf
					num++
					fmt.Println(mediaName, "rtcp pusher recieved data number =", num)

				} else {
					fmt.Printf("error occured when read from pusher = %v\n", err)
					return
				}
			}
		}()
	}
	if session.SessionClientType == PullerClient {
		go func() {
			var num int = 0
			for !session.IfStop {
				for session.IfPause {
					time.Sleep(time.Duration(10) * time.Millisecond)
				}
				data := <-session.RtpPackageChannel
				if _, err := session.RtpUDPConnToPuller.Write(*data); err != nil {
					fmt.Printf("error occured when write to puller = %v\n", err)
					return
				}
				time.Sleep(time.Duration(10) * time.Millisecond)
				num++
				if num%200 == 0 {
					fmt.Println(mediaName, "rtp puller sended data number =", num)
				}
			}
		}()
		go func() {
			var num int = 0
			for !session.IfStop {
				for session.IfPause {
					time.Sleep(time.Duration(10) * time.Millisecond)
				}
				data := <-session.RtcpPackageChannel
				if _, err := session.RtcpUDPConnToPuller.Write(*data); err != nil {
					fmt.Printf("error occured when write to puller error = %v\n", err)
					return
				}
				time.Sleep(time.Duration(40) * time.Millisecond)
				num++
				fmt.Println(mediaName, "rctp puller sended data number =", num)
			}
		}()
	}
	return nil
}

//PauseTransfer pause this transfer
func (session *RtpRtcpSession) PauseTransfer() error {
	session.IfPause = true
	return nil
}

//StopTransfer stop this transfer
func (session *RtpRtcpSession) StopTransfer() error {
	session.IfStop = true
	return nil
}
