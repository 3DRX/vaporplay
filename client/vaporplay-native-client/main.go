package main

import (
	"image"

	"github.com/3DRX/vaporplay/client/vaporplay-native-client/peerconnection"
	"github.com/3DRX/vaporplay/client/vaporplay-native-client/signaling"
	"github.com/3DRX/vaporplay/client/vaporplay-native-client/ui"
	"github.com/pion/webrtc/v4"
)

func main() {
	sdpChan := make(chan webrtc.SessionDescription)
	sdpReplyChan := make(chan webrtc.SessionDescription)
	candidateChan := make(chan webrtc.ICECandidateInit)
	frameChan := make(chan image.Image, 120)

	uiThread, startGamePromise := ui.NewUIThread(frameChan)
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
			frameChan,
		)
		go peerconnectionThread.Spin()
	}()
	uiThread.Spin()
}
