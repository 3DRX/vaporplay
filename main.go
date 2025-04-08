package main

import (
	"embed"
	"flag"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/3DRX/vaporplay/config"
	"github.com/3DRX/vaporplay/peerconnection"
	"github.com/3DRX/vaporplay/signaling"
	"github.com/pion/webrtc/v4"
)

//go:embed tmp/webui/*
var embedFS embed.FS

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

	subFS, err := fs.Sub(embedFS, "tmp/webui")
	if err != nil {
		panic(err)
	}

	signalingThread := signaling.NewSignalingThread(
		cfg,
		sendSDPChan,
		recvSDPChan,
		sendCandidateChan,
		recvCandidateChan,
		http.FS(subFS),
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
