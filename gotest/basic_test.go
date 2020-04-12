package gotest

import (
	"testing"

	"github.com/darunshen/stmpt/rtsp"
)

const (
	//ReadBuffer bio&tcp&udp read buffer size
	ReadBuffer int = 10485760
	//WriteBuffer bio&tcp&udp write buffer size
	WriteBuffer int = 10485760
	//PushChannelBuffer pusher channel buffer size
	PushChannelBuffer int = 1
	//PullChannelBuffer puller channel buffer size
	PullChannelBuffer int = 1
)

func TestWriteReadRtspPackage(t *testing.T) {
	server := rtsp.Server{}
	server.Start("0.0.0.0:2333",
		ReadBuffer, WriteBuffer, PushChannelBuffer, PullChannelBuffer)
}
