package main

import (
	"github.com/3DRX/vaporplay/client/vaporplay-native-client/peerconnection"
	"github.com/3DRX/vaporplay/client/vaporplay-native-client/signaling"
	"github.com/3DRX/vaporplay/client/vaporplay-native-client/ui"
	"github.com/pion/webrtc/v4"
)

func main() {
	sdpChan := make(chan webrtc.SessionDescription)
	sdpReplyChan := make(chan webrtc.SessionDescription)
	candidateChan := make(chan webrtc.ICECandidateInit)

	uiThread := ui.NewUIThread()
	startGamePromise := uiThread.Spin()
	clientCfg := <-startGamePromise
	signalingThread := signaling.NewSignalingThread(
		clientCfg,
		sdpChan,
		sdpReplyChan,
		candidateChan,
	)
	peerconnectionThread := peerconnection.NewPeerConnectionThread(
		clientCfg,
		sdpChan,
		sdpReplyChan,
		candidateChan,
		signalingThread.SignalCandidate,
	)
	go signalingThread.Spin()
	go peerconnectionThread.Spin()
	select {}
}
