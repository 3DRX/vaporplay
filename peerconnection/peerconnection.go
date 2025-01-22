package peerconnection

import (
	"log/slog"
	"os"

	_ "github.com/pion/mediadevices/pkg/driver/camera"
	"github.com/pion/mediadevices/pkg/prop"

	// TODO: custom adapter for game window
	// rosmediadevicesadapter "github.com/3DRX/webrtc-ros-bridge/ros_mediadevices_adapter"
	"github.com/pion/interceptor"
	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/codec/vpx"
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
}

func NewPeerConnectionThread(
	sendSDPChan chan<- webrtc.SessionDescription,
	recvSDPChan <-chan webrtc.SessionDescription,
	sendCandidateChan chan<- webrtc.ICECandidateInit,
	recvCandidateChan <-chan webrtc.ICECandidateInit,
) *PeerConnectionThread {
	// TODO
	// rosmediadevicesadapter.Initialize(imgChan, imgWidth, imgHeight, frameRate)
	vp8Params, err := vpx.NewVP8Params()
	if err != nil {
		panic(err)
	}
	vp8Params.BitRate = 5_000_000
	codecselector := mediadevices.NewCodecSelector(
		mediadevices.WithVideoEncoders(&vp8Params),
	)
	m := &webrtc.MediaEngine{}
	codecselector.Populate(m)
	i := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
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

	pc := &PeerConnectionThread{
		sendSDPChan:       sendSDPChan,
		recvSDPChan:       recvSDPChan,
		sendCandidateChan: sendCandidateChan,
		recvCandidateChan: recvCandidateChan,
		peerConnection:    peerConnection,
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
			// TODO: restore state to be able to connect with a client again
			os.Exit(0)
		}
	})
	go pc.handleRemoteICECandidate()
	pc.sendSDPChan <- offer
	remoteSDP := <-pc.recvSDPChan
	pc.peerConnection.SetRemoteDescription(remoteSDP)
}
