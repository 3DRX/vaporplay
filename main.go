package main

import (
	"github.com/3DRX/piongs/config"
	"github.com/3DRX/piongs/peerconnection"
	"github.com/3DRX/piongs/signaling"
	"github.com/pion/webrtc/v4"
)

func main() {
	cfg := config.LoadCfg()
	sendSDPChan := make(chan webrtc.SessionDescription)
	recvSDPChan := make(chan webrtc.SessionDescription)
	sendCandidateChan := make(chan webrtc.ICECandidateInit)
	recvCandidateChan := make(chan webrtc.ICECandidateInit)

	signalingThread := signaling.NewSignalingThread(
		cfg,
		sendSDPChan,
		recvSDPChan,
		sendCandidateChan,
		recvCandidateChan,
	)
	haveReceiverPromise := signalingThread.Spin()
	<-haveReceiverPromise

	peerConnectionThread := peerconnection.NewPeerConnectionThread(
		sendSDPChan,
		recvSDPChan,
		sendCandidateChan,
		recvCandidateChan,
	)
	go peerConnectionThread.Spin()
	select {}
}
