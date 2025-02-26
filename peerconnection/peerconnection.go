package peerconnection

import (
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/3DRX/piongs/codec/ffmpeg"
	"github.com/3DRX/piongs/config"
	"github.com/3DRX/piongs/gamecapture"
	"github.com/pion/interceptor"

	// "github.com/pion/interceptor/pkg/cc"
	// "github.com/pion/interceptor/pkg/gcc"
	"github.com/pion/mediadevices"
	// "github.com/pion/mediadevices/pkg/codec/vpx"
	"github.com/pion/mediadevices/pkg/prop"
	"github.com/pion/webrtc/v4"
)

type AddStreamAction struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}

type AddVideoTrackAction struct {
	Type     string `json:"type"`
	Id       string `json:"id"`
	StreamId string `json:"stream_id"`
	SrcId    string `json:"src"`
}

type PeerConnectionThread struct {
	sendSDPChan       chan<- webrtc.SessionDescription
	recvSDPChan       <-chan webrtc.SessionDescription
	sendCandidateChan chan<- webrtc.ICECandidateInit
	recvCandidateChan <-chan webrtc.ICECandidateInit
	peerConnection    *webrtc.PeerConnection
	gameConfig        *config.GameConfig
	gamepadControl    *GamepadControl
}

func NewPeerConnectionThread(
	sendSDPChan chan<- webrtc.SessionDescription,
	recvSDPChan <-chan webrtc.SessionDescription,
	sendCandidateChan chan<- webrtc.ICECandidateInit,
	recvCandidateChan <-chan webrtc.ICECandidateInit,
	selectedGame *config.GameConfig,
) *PeerConnectionThread {
	params, err := ffmpeg.NewH264Params()
	if err != nil {
		panic(err)
	}
	params.BitRate = 20_000_000
	codecselector := mediadevices.NewCodecSelector(
		mediadevices.WithVideoEncoders(&params),
	)
	m := &webrtc.MediaEngine{}
	codecselector.Populate(m)
	i := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		panic(err)
	}
	// congestionControllerFactory, err := cc.NewInterceptor(func() (cc.BandwidthEstimator, error) {
	// 	return gcc.NewSendSideBWE(gcc.SendSideBWEInitialBitrate(400_000))
	// })
	// if err != nil {
	// 	panic(err)
	// }
	// estimatorChan := make(chan cc.BandwidthEstimator, 1)
	// congestionControllerFactory.OnNewPeerConnection(func(id string, estimator cc.BandwidthEstimator) { //nolint: revive
	// 	estimatorChan <- estimator
	// })
	// i.Add(congestionControllerFactory)
	// if err := webrtc.ConfigureTWCCHeaderExtensionSender(m, i); err != nil {
	// 	panic(err)
	// }
	// if err := webrtc.ConfigureCongestionControlFeedback(m, i); err != nil {
	// 	panic(err)
	// }
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i))
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}
	peerConnection, err := api.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}
	slog.Info("Created peer connection")

	// estimator := <-estimatorChan

	// // estimator.GetTargetBitrate()
	// estimator.OnTargetBitrateChange(func(bitrate int) {
	// 	slog.Info("Target bitrate changed", "bitrate", bitrate)
	// })

	gamecapture.Initialize(selectedGame)

	mediaStream, err := mediadevices.GetUserMedia(mediadevices.MediaStreamConstraints{
		Video: func(constraint *mediadevices.MediaTrackConstraints) {
			constraint.Width = prop.Int(1920)
			constraint.Height = prop.Int(1080)
			constraint.FrameRate = prop.Float(60)
		},
		Codec: codecselector,
	})
	if err != nil {
		panic(err)
	}
	for _, videoTrack := range mediaStream.GetVideoTracks() {
		videoTrack.OnEnded(func(err error) {
			slog.Error("Track ended", "error", err)
		})
		_, err := peerConnection.AddTransceiverFromTrack(
			videoTrack,
			webrtc.RTPTransceiverInit{
				Direction: webrtc.RTPTransceiverDirectionSendonly,
			},
		)
		if err != nil {
			panic(err)
		}
		slog.Info("add video track success")
	}

	gamepadControl, err := NewGamepadControl()
	if err != nil {
		panic(err)
	}

	pc := &PeerConnectionThread{
		sendSDPChan:       sendSDPChan,
		recvSDPChan:       recvSDPChan,
		sendCandidateChan: sendCandidateChan,
		recvCandidateChan: recvCandidateChan,
		peerConnection:    peerConnection,
		gameConfig:        selectedGame,
		gamepadControl:    gamepadControl,
	}
	return pc
}

func (pc *PeerConnectionThread) handleRemoteICECandidate() {
	for {
		candidate := <-pc.recvCandidateChan
		if err := pc.peerConnection.AddICECandidate(candidate); err != nil {
			panic(err)
		}
	}
}

func (pc *PeerConnectionThread) Spin() {
	datachannel, err := pc.peerConnection.CreateDataChannel("controller", nil)
	if err != nil {
		panic(err)
	}
	datachannel.OnOpen(func() {
		slog.Info("datachannel open", "label", datachannel.Label(), "ID", datachannel.ID())
	})
	datachannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		dto := &GamepadDTO{}
		err := json.Unmarshal(msg.Data, dto)
		if err != nil {
			slog.Warn("Failed to unmarshal datachannel message", "error", err)
		}
		slog.Info("datachannel message", "data", dto)
		pc.gamepadControl.SendGamepadState(dto)
	})

	offer, err := pc.peerConnection.CreateOffer(nil)
	if err != nil {
		panic(err)
	}
	pc.peerConnection.SetLocalDescription(offer)
	pc.peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		pc.sendCandidateChan <- c.ToJSON()
	})
	pc.peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		if s == webrtc.PeerConnectionStateClosed {
			slog.Info("Peer connection closed")
			// kill game processes
			for _, processConfig := range pc.gameConfig.EndGameCommands {
				args := []string{"killall"}
				if len(processConfig.Flags) != 0 {
					args = append(args, processConfig.Flags...)
				} else {
					args = append(args, "-v", "-w")
				}
				args = append(args, processConfig.ProcessName)
				// print command
				slog.Info("Killing game process", "command", strings.Join(args, " "))
				cmd := exec.Command(args[0], args[1:]...)
				_, err := cmd.Output()
				if err != nil {
					slog.Error("Failed to kill game process", "error", err)
					continue
				}
			}
			// TODO: restore state to be able to connect with a client again
			os.Exit(0)
		}
	})
	go pc.handleRemoteICECandidate()
	pc.sendSDPChan <- offer
	remoteSDP := <-pc.recvSDPChan
	pc.peerConnection.SetRemoteDescription(remoteSDP)
}
