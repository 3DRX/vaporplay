package peerconnection

import (
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"runtime/pprof"
	"strings"

	"github.com/3DRX/piongs/codec/ffmpeg"
	"github.com/3DRX/piongs/config"
	"github.com/3DRX/piongs/gamecapture"
	"github.com/3DRX/piongs/interceptor/cc"
	"github.com/3DRX/piongs/interceptor/gcc"
	"github.com/asticode/go-astiav"
	"github.com/pion/interceptor"

	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/codec"
	"github.com/pion/mediadevices/pkg/driver"
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
	estimatorChan     chan cc.BandwidthEstimator
	cpuProfile        string
	videoDriverLabel  string
}

func NewPeerConnectionThread(
	sendSDPChan chan<- webrtc.SessionDescription,
	recvSDPChan <-chan webrtc.SessionDescription,
	sendCandidateChan chan<- webrtc.ICECandidateInit,
	recvCandidateChan <-chan webrtc.ICECandidateInit,
	selectedGame *config.GameConfig,
	cpuProfile string,
) *PeerConnectionThread {
	params, err := ffmpeg.NewAV1NVENCParams(
		"/dev/dri/card1",
		astiav.PixelFormat(astiav.PixelFormatBgra),
	)
	if err != nil {
		panic(err)
	}
	params.BitRate = 5_000_000
	params.FrameRate = 90
	params.KeyFrameInterval = -1
	codecselector := mediadevices.NewCodecSelector(
		mediadevices.WithVideoEncoders(&params),
	)
	m := &webrtc.MediaEngine{}
	codecselector.Populate(m)
	i := &interceptor.Registry{}
	// pacer := gcc.NewLeakyBucketPacer(1_000_000 * 1.5)
	pacer := gcc.NewNoOpPacer()
	congestionControllerFactory, err := cc.NewInterceptor(func() (cc.BandwidthEstimator, error) {
		return gcc.NewSendSideBWE(
			gcc.SendSideBWEInitialBitrate(params.BitRate),
			gcc.SendSideBWEMaxBitrate(30_000_000),
			gcc.SendSideBWEMinBitrate(500_000),
			gcc.SendSideBWEPacer(pacer),
		)
	})
	if err != nil {
		panic(err)
	}
	estimatorChan := make(chan cc.BandwidthEstimator, 1)
	congestionControllerFactory.OnNewPeerConnection(func(id string, estimator cc.BandwidthEstimator) { //nolint: revive
		estimatorChan <- estimator
	})
	i.Add(congestionControllerFactory)
	if err := webrtc.ConfigureNack(m, i); err != nil {
		panic(err)
	}
	if err := webrtc.ConfigureTWCCHeaderExtensionSender(m, i); err != nil {
		panic(err)
	}
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

	videoDriverLabel := gamecapture.Initialize(selectedGame)

	mediaStream, err := mediadevices.GetUserMedia(mediadevices.MediaStreamConstraints{
		Video: func(constraint *mediadevices.MediaTrackConstraints) {
			constraint.Width = prop.Int(1920)
			constraint.Height = prop.Int(1080)
			constraint.FrameRate = prop.Float(params.FrameRate)
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
		estimatorChan:     estimatorChan,
		cpuProfile:        cpuProfile,
		videoDriverLabel:  videoDriverLabel,
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
	endSpinPromise := make(chan struct{})
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
		// slog.Info("datachannel message", "data", dto)
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
		switch s {
		case webrtc.PeerConnectionStateConnected:
			senders := pc.peerConnection.GetSenders()
			var bitrateController codec.BitRateController
			for _, sender := range senders {
				vt, ok := sender.Track().(*mediadevices.VideoTrack)
				if !ok {
					continue
				}
				encoderController := vt.EncoderController()
				bitrateController, ok = encoderController.(codec.BitRateController)
				if !ok {
					bitrateController = nil
					slog.Warn("current codec does not implement BitRateController")
				}
			}
			estimator := <-pc.estimatorChan
			estimator.OnTargetBitrateChange(func(bitrate int) {
				if bitrateController == nil {
					slog.Warn("bitrate controller is nil")
					return
				}
				slog.Info("setting bitrate", "bitrate", bitrate)
				bitrateController.SetBitRate(int(float64(bitrate) * 0.9))
			})
		case webrtc.PeerConnectionStateClosed:
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
			endSpinPromise <- struct{}{}
		}
	})
	go pc.handleRemoteICECandidate()
	pc.sendSDPChan <- offer
	remoteSDP := <-pc.recvSDPChan
	if pc.cpuProfile != "" {
		f, err := os.Create(pc.cpuProfile)
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	pc.peerConnection.SetRemoteDescription(remoteSDP)
	select {
	case <-endSpinPromise:
		// close all driver and encoder
		if err := pc.gamepadControl.Close(); err != nil {
			slog.Error("failed to close gamepad control", "error", err)
			panic(err)
		}
		drivers := driver.GetManager().Query(func(d driver.Driver) bool {
			if d.Info().Label == pc.videoDriverLabel {
				return true
			}
			return false
		})
		if len(drivers) == 0 {
			slog.Warn("no driver to close")
		}
		for _, d := range drivers {
			if err := d.Close(); err != nil {
				slog.Error("failed to close driver "+d.Info().Label, "error", err)
				panic(err)
			}
		}
		transceivers := pc.peerConnection.GetTransceivers()
		for _, t := range transceivers {
			if err := t.Stop(); err != nil {
				slog.Error("failed to stop transceiver", "error", err)
				panic(err)
			}
		}
		if err := pc.peerConnection.GracefulClose(); err != nil {
			slog.Error("failed to close peer connection", "error", err)
			panic(err)
		}
		slog.Info("peer connection thread closed")
	}
}
