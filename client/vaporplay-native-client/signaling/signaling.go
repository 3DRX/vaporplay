package signaling

import (
	"encoding/json"
	"log/slog"
	"net/url"

	clientconfig "github.com/3DRX/vaporplay/client/vaporplay-native-client/client-config"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

type SignalingThread struct {
	cfg           *clientconfig.ClientConfig
	c             *websocket.Conn
	sdpChan       chan<- webrtc.SessionDescription
	sdpReplyChan  <-chan webrtc.SessionDescription
	candidateChan chan<- webrtc.ICECandidateInit
}

func NewSignalingThread(
	sdpChan chan webrtc.SessionDescription,
	sdpReplyChan <-chan webrtc.SessionDescription,
	candidateChan chan<- webrtc.ICECandidateInit,
) *SignalingThread {
	return &SignalingThread{
		c:             nil,
		sdpChan:       sdpChan,
		sdpReplyChan:  sdpReplyChan,
		candidateChan: candidateChan,
	}
}

func (s *SignalingThread) Spin(cfg *clientconfig.ClientConfig) {
	s.cfg = cfg
	u := url.URL{Scheme: "ws", Host: s.cfg.Addr, Path: "/webrtc"}
	slog.Info("start spinning", "url", u.String())
	wsConn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		panic(err)
	}
	s.c = wsConn
	defer wsConn.Close()
	go func() {
		for {
			_, message, err := wsConn.ReadMessage()
			if err != nil {
				slog.Error("recv error", "err", err)
				return
			}
			s.onWsMessage(message)
		}
	}()

	cfgMessage, err := json.Marshal(s.cfg.SessionConfig)
	if err != nil {
		slog.Error("compose session config error", "error", err)
		return
	}
	wsConn.WriteMessage(websocket.TextMessage, cfgMessage)
	slog.Info("send configure message")
	answer := <-s.sdpReplyChan // await answer from peer connection
	payload, err := json.Marshal(answer)
	if err != nil {
		slog.Error("marshal error", "error", err)
	}
	wsConn.WriteMessage(websocket.TextMessage, payload)
	slog.Info("send answer", "sdp", answer.SDP)

	select {}
}

func (s *SignalingThread) onWsMessage(messageRaw []byte) {
	// see if this is webrtc.SessionDescription or webrtc.ICECandidateInit
	sdp := webrtc.SessionDescription{}
	candidate := webrtc.ICECandidateInit{}
	err := json.Unmarshal(messageRaw, &sdp)
	if err == nil && sdp.SDP != "" {
		s.sdpChan <- sdp
		slog.Info("received SDP", "sdp", sdp)
		return
	}
	err = json.Unmarshal(messageRaw, &candidate)
	if err == nil && candidate.Candidate != "" {
		s.candidateChan <- candidate
		slog.Info("received ICE candidate", "candidate", candidate)
		return
	}
	slog.Warn("unknown message", "message", string(messageRaw))
}
