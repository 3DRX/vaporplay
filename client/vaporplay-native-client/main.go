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

	uiThread, startGamePromise := ui.NewUIThread()
	signalingThread := signaling.NewSignalingThread(
		sdpChan,
		sdpReplyChan,
		candidateChan,
	)
	go func() {
		clientCfg := <-startGamePromise
		go signalingThread.Spin(clientCfg)
		peerconnectionThread := peerconnection.NewPeerConnectionThread(
			clientCfg,
			sdpChan,
			sdpReplyChan,
			candidateChan,
			signalingThread.SignalCandidate,
		)
		go peerconnectionThread.Spin()
	}()
	uiThread.Spin()
}
