package main

import (
	"flag"
	"log/slog"

	"github.com/3DRX/piongs/config"
	"github.com/3DRX/piongs/peerconnection"
	"github.com/3DRX/piongs/signaling"
	"github.com/pion/webrtc/v4"
)

var cpuProfile = flag.String("cpuprofile", "", "write cpu profile to file")
var configPath = flag.String("config", "", "path to config file")

func main() {
	flag.Parse()
	if *configPath == "" {
		panic("config file path is required")
	}
	cfg := config.LoadCfg(*configPath)
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
	sessionConfig := <-haveReceiverPromise

	peerConnectionThread := peerconnection.NewPeerConnectionThread(
		sendSDPChan,
		recvSDPChan,
		sendCandidateChan,
		recvCandidateChan,
		cfg,
		sessionConfig,
		*cpuProfile,
	)
	peerConnectionThread.Spin()

	// cleaning
	if err := signalingThread.Close(); err != nil {
		slog.Error("failed to close signaling thread", "error", err)
		panic(err)
	}
	slog.Info("signaling thread closed")
}
