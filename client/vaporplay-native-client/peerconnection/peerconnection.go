package peerconnection

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/3DRX/vaporplay/gamepaddto"
	"github.com/3DRX/vaporplay/interceptor/nack"
	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
)

type PeerConnectionThread struct {
	sdpChan         <-chan webrtc.SessionDescription
	sdpReplyChan    chan<- webrtc.SessionDescription
	candidateChan   <-chan webrtc.ICECandidateInit
	peerConnection  *webrtc.PeerConnection
	signalCandidate func(c webrtc.ICECandidateInit) error
}

func NewPeerConnectionThread(
	sdpChan chan webrtc.SessionDescription,
	sdpReplyChan chan<- webrtc.SessionDescription,
	candidateChan <-chan webrtc.ICECandidateInit,
	signalCandidate func(c webrtc.ICECandidateInit) error,
) *PeerConnectionThread {
	m := &webrtc.MediaEngine{}
	i := &interceptor.Registry{}
	s := webrtc.SettingEngine{}
	if err := m.RegisterCodec(
		webrtc.RTPCodecParameters{
			RTPCodecCapability: webrtc.RTPCodecCapability{
				MimeType:    webrtc.MimeTypeH264,
				ClockRate:   90000,
				Channels:    0,
				SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f",
				RTCPFeedback: []webrtc.RTCPFeedback{
					{Type: "nack", Parameter: ""},
					{Type: "nack", Parameter: "pli"},
				},
			},
			PayloadType: 112,
		},
		webrtc.RTPCodecTypeVideo,
	); err != nil {
		panic(err)
	}
	if err := m.RegisterCodec(
		webrtc.RTPCodecParameters{
			RTPCodecCapability: webrtc.RTPCodecCapability{
				MimeType:     webrtc.MimeTypeRTX,
				ClockRate:    90000,
				Channels:     0,
				SDPFmtpLine:  "apt=112",
				RTCPFeedback: nil,
			},
			PayloadType: 113,
		},
		webrtc.RTPCodecTypeVideo,
	); err != nil {
		panic(err)
	}

	// if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
	// 	panic(err)
	// }
	nackGenerator, err := nack.NewGeneratorInterceptor()
	if err != nil {
		panic(err)
	}
	i.Add(nackGenerator)

	s.SetFireOnTrackBeforeFirstRTP(true)

	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(m),
		webrtc.WithInterceptorRegistry(i),
		webrtc.WithSettingEngine(s),
	)
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback,
	}
	peerConnection, err := api.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}
	slog.Info("Created peer connection")
	return &PeerConnectionThread{
		sdpChan:         sdpChan,
		sdpReplyChan:    sdpReplyChan,
		candidateChan:   candidateChan,
		peerConnection:  peerConnection,
		signalCandidate: signalCandidate,
	}
}

func handleSignalingMessage(pc *PeerConnectionThread) {
	for {
		select {
		case sdp := <-pc.sdpChan:
			slog.Info("received SDP", "sdp", sdp.SDP)
			err := pc.peerConnection.SetRemoteDescription(sdp)
			if err != nil {
				panic(err)
			}
			answer, err := pc.peerConnection.CreateAnswer(nil)
			if err != nil {
				panic(err)
			}
			pc.sdpReplyChan <- answer
			err = pc.peerConnection.SetLocalDescription(answer)
			if err != nil {
				panic(err)
			}
		case candidate := <-pc.candidateChan:
			err := pc.peerConnection.AddICECandidate(candidate)
			if err != nil {
				panic(err)
			}
			slog.Info("received ICE candidate", "candidate", candidate)
		}
	}
}

func (pc *PeerConnectionThread) Spin() {
	videoDecoder := newVideoDecoder()
	videoDecoder.Init()
	pc.peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		slog.Info("OnConnectionStateChange", "state", state.String())
	})
	pc.peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		if err := pc.signalCandidate(c.ToJSON()); err != nil {
			panic(err)
		}
	})
	pc.peerConnection.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		slog.Info("PeerConnectionChannel: received track", "track", track.ID())
		if track.Kind() == webrtc.RTPCodecTypeVideo {
			// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
			go func() {
				ticker := time.NewTicker(time.Second * 3)
				for range ticker.C {
					errSend := pc.peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}})
					if errSend != nil {
						fmt.Println(errSend)
					}
				}
			}()
		}
		for {
			rtp, _, readErr := track.ReadRTP()
			if readErr != nil {
				panic(readErr)
			}
			videoDecoder.PushPacket(rtp)
		}
	})
	pc.peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		if d.Label() == "controller" {
			d.OnOpen(func() {
				slog.Info("datachannel open", "label", d.Label(), "ID", d.ID())
			})
			dto := gamepaddto.GamepadDTO{}
			dtoString, err := json.Marshal(dto)
			if err != nil {
				slog.Warn("GamepadDTO marshal error", "error", err)
			}
			d.SendText(string(dtoString))
		}
	})
	go handleSignalingMessage(pc)
}
