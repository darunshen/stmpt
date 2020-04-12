package rtsp

import (
	"container/list"
	"fmt"
	"sync"

	"gortc.io/sdp"
)

//PusherPullersPair one pusher maps multiple pullers
type PusherPullersPair struct {
	Pusher          *RtpRtcpSession
	Pullers         *list.List
	PullersMutex    sync.Mutex
	rtpPackageChan  chan RtpRtcpPackage
	rtcpPackageChan chan RtpRtcpPackage
	IfStop          bool
}

//PusherPullersSession session includes pusher and pullers
type PusherPullersSession struct {
	PusherPullersPairMap map[MediaType]*PusherPullersPair // has vidio and audio
	SdpMessage           *sdp.Message                     // sdp info from pusher
	SdpContent           *string                          // sdp raw content
	AudioStreamName      *string                          // audio stream name from sdp content
	VideoStreamName      *string                          // video stream name from sdp content
}

//AddRtpRtcpSession add a rtp-rtcp-session to this pusher-pullers-session
func (session *PusherPullersSession) AddRtpRtcpSession(
	clientType ClientType, mediaType MediaType,
	rtpPort, rtcpPort, remoteIP *string, rtspSessionID string) error {
	if session.PusherPullersPairMap == nil {
		session.PusherPullersPairMap = make(map[MediaType]*PusherPullersPair)
	}
	switch clientType {
	case PusherClient:
		if _, ok := session.PusherPullersPairMap[mediaType]; ok {
			return fmt.Errorf("pusher's request's url resource already used")
		}
		ppp := new(PusherPullersPair)
		ppp.rtpPackageChan = make(chan RtpRtcpPackage, PushChannelBufferSize)
		ppp.rtcpPackageChan = make(chan RtpRtcpPackage, PullChannelBufferSize)
		ppp.Pullers = list.New()
		session.PusherPullersPairMap[mediaType] = ppp
		rrs := new(RtpRtcpSession)
		if err := rrs.StartRtpRtcpSession(clientType, mediaType, nil, rtspSessionID); err != nil {
			return err
		}
		if err := ppp.StartDispatch(); err != nil {
			return err
		}
		ppp.Pusher = rrs
		*rtpPort = *rrs.RtpServerPort
		*rtcpPort = *rrs.RtcpServerPort
	case PullerClient:
		ppp, ok := session.PusherPullersPairMap[mediaType]
		if !ok {
			return fmt.Errorf("puller's request's url resource not found")
		}
		rrs := new(RtpRtcpSession)
		if err := rrs.StartRtpRtcpSession(clientType, mediaType, &PullerClientInfo{
			RtpRemotePort:  rtpPort,
			RtcpRemotePort: rtcpPort,
			IPRemote:       remoteIP,
		}, rtspSessionID); err != nil {
			return err
		}
		ppp.PullersMutex.Lock()
		ppp.Pullers.PushBack(rrs)
		ppp.PullersMutex.Unlock()
	default:
		return fmt.Errorf("clientType error : not support")
	}
	return nil
}

//StartSession start goroutines(rtp/rtcp) created by rtspSessionID
func (session *PusherPullersSession) StartSession(rtspSessionID *string) []error {
	returnErr := make([]error, 0)
	for _, ppp := range session.PusherPullersPairMap {
		if err := ppp.Start(rtspSessionID); len(err) != 0 {
			returnErr = append(returnErr, err...)
		}
	}
	return returnErr
}

//PauseSession Pause goroutines(rtp/rtcp) created by rtspSessionID
func (session *PusherPullersSession) PauseSession(rtspSessionID *string) []error {
	returnErr := make([]error, 0)
	for _, ppp := range session.PusherPullersPairMap {
		if err := ppp.Pause(rtspSessionID); len(err) != 0 {
			returnErr = append(returnErr, err...)
		}
	}
	return returnErr
}

//StopSession stop goroutines(rtp/rtcp) created by rtspSessionID
func (session *PusherPullersSession) StopSession(rtspSessionID *string) []error {
	returnErr := make([]error, 0)
	for _, ppp := range session.PusherPullersPairMap {
		if err := ppp.Stop(rtspSessionID); len(err) != 0 {
			returnErr = append(returnErr, err...)
		}
	}
	return returnErr
}

//TCPPackageProcess transfer rtp/rtcp package throw tcp conn
func (session *PusherPullersSession) TCPPackageProcess() error {
	return nil
}

//StartDispatch begin package(rtp/rtcp) dispatch from pusher to pullers
func (session *PusherPullersPair) StartDispatch() error {
	go func() {
		for !session.IfStop {
			data := <-session.rtpPackageChan
			var next *list.Element
			for puller := session.Pullers.Front(); puller != nil; puller = next {
				next = puller.Next()
				if puller.Value.(*RtpRtcpSession).IfStop {
					session.Pullers.Remove(puller)
					fmt.Println("rtp session deleted,rtsp session id =" +
						puller.Value.(*RtpRtcpSession).RtspSessionID +
						",session.Pullers size = " + string(session.Pullers.Len()))
				} else {
					puller.Value.(*RtpRtcpSession).RtpPackageChannel <- &data
				}
			}
		}
	}()
	go func() {
		for !session.IfStop {
			data := <-session.rtcpPackageChan
			var next *list.Element
			for puller := session.Pullers.Front(); puller != nil; puller = next {
				next = puller.Next()
				if puller.Value.(*RtpRtcpSession).IfStop {
					session.Pullers.Remove(puller)
					fmt.Println("rtcp session deleted,rtsp session id =" +
						puller.Value.(*RtpRtcpSession).RtspSessionID +
						",session.Pullers size = " + string(session.Pullers.Len()))
				} else {
					puller.Value.(*RtpRtcpSession).RtcpPackageChannel <- &data
				}
			}
		}
	}()
	return nil
}

//Start start package(rtp/rtcp) transfer from pusher to pullers
func (session *PusherPullersPair) Start(rtspSessionID *string) []error {
	returnErr := make([]error, 0)
	if session.Pusher.RtspSessionID == *rtspSessionID {
		if err := session.Pusher.BeginTransfer(
			session.rtpPackageChan, session.rtcpPackageChan); err != nil {
			returnErr = append(returnErr, err)
		}
		for puller := session.Pullers.Front(); puller != nil; puller = puller.Next() {
			if err := puller.Value.(*RtpRtcpSession).BeginTransfer(nil, nil); err != nil {
				returnErr = append(returnErr, err)
			}
		}
	} else {
		var find bool = false
		for puller := session.Pullers.Front(); puller != nil; puller = puller.Next() {
			if puller.Value.(*RtpRtcpSession).RtspSessionID == *rtspSessionID {
				if err := puller.Value.(*RtpRtcpSession).BeginTransfer(nil, nil); err != nil {
					returnErr = append(returnErr, err)
				}
				find = true
				break
			}
		}
		if !find {
			returnErr = append(returnErr,
				fmt.Errorf("PusherPullersPair Start error : not find the session id"))
		}
	}
	return returnErr
}

//Pause pause package(rtp/rtcp) transfer from pusher to pullers
func (session *PusherPullersPair) Pause(rtspSessionID *string) []error {
	returnErr := make([]error, 0)
	if session.Pusher.RtspSessionID == *rtspSessionID {
		if err := session.Pusher.PauseTransfer(); err != nil {
			returnErr = append(returnErr, err)
		}
		for puller := session.Pullers.Front(); puller != nil; puller = puller.Next() {
			if err := puller.Value.(*RtpRtcpSession).PauseTransfer(); err != nil {
				returnErr = append(returnErr, err)
			}
		}
	} else {
		var find bool = false
		for puller := session.Pullers.Front(); puller != nil; puller = puller.Next() {
			if puller.Value.(*RtpRtcpSession).RtspSessionID == *rtspSessionID {
				if err := puller.Value.(*RtpRtcpSession).PauseTransfer(); err != nil {
					returnErr = append(returnErr, err)
				}
				find = true
				break
			}
		}
		if !find {
			returnErr = append(returnErr,
				fmt.Errorf("PusherPullersPair Pause error : not find the session id"))
		}
	}
	return returnErr
}

//Stop stop package(rtp/rtcp) transfer from pusher to pullers
func (session *PusherPullersPair) Stop(rtspSessionID *string) []error {
	returnErr := make([]error, 0)
	if session.Pusher.RtspSessionID == *rtspSessionID {
		if err := session.Pusher.StopTransfer(); err != nil {
			returnErr = append(returnErr, err)
		}
		for puller := session.Pullers.Front(); puller != nil; puller = puller.Next() {
			if err := puller.Value.(*RtpRtcpSession).StopTransfer(); err != nil {
				returnErr = append(returnErr, err)
			}
		}
		session.IfStop = true
	} else {
		var find bool = false
		for puller := session.Pullers.Front(); puller != nil; puller = puller.Next() {
			if puller.Value.(*RtpRtcpSession).RtspSessionID == *rtspSessionID {
				if err := puller.Value.(*RtpRtcpSession).StopTransfer(); err != nil {
					returnErr = append(returnErr, err)
				}
				find = true
				break
			}
		}
		if !find {
			returnErr = append(returnErr,
				fmt.Errorf("PusherPullersPair Stop error : not find the session id"))
		}
	}

	return returnErr
}
