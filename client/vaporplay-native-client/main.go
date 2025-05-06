package main

import (
	"flag"
	"image"

	"github.com/3DRX/vaporplay/client/vaporplay-native-client/peerconnection"
	"github.com/3DRX/vaporplay/client/vaporplay-native-client/signaling"
	"github.com/3DRX/vaporplay/client/vaporplay-native-client/ui"
	"github.com/pion/webrtc/v4"
)

// TODO: add profile capability to native client
// var cpuProfile = flag.String("cpuprofile", "", "write cpu profile to file")
var configPath = flag.String("config", "client_config.json", "path to config file")

func main() {
	flag.Parse()
	if *configPath == "" {
		panic("config file path is required")
	}
	sdpChan := make(chan webrtc.SessionDescription)
	sdpReplyChan := make(chan webrtc.SessionDescription)
	candidateChan := make(chan webrtc.ICECandidateInit)
	frameChan := make(chan image.Image, 120)
	closeWindowPromise := make(chan struct{}, 1)

	uiThread, startGamePromise := ui.NewUIThread(
		frameChan,
		configPath,
		closeWindowPromise,
	)
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
			closeWindowPromise,
		)
		go peerconnectionThread.Spin()
	}()
	uiThread.Spin()
}
