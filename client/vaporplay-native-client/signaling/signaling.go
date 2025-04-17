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
	recv          chan []byte
	c             *websocket.Conn
	sdpChan       chan<- webrtc.SessionDescription
	sdpReplyChan  <-chan webrtc.SessionDescription
	candidateChan chan<- webrtc.ICECandidateInit
}

func NewSignalingThread(
	cfg *clientconfig.ClientConfig,
	sdpChan chan webrtc.SessionDescription,
	sdpReplyChan <-chan webrtc.SessionDescription,
	candidateChan chan<- webrtc.ICECandidateInit,
) *SignalingThread {
	return &SignalingThread{
		cfg:           cfg,
		recv:          make(chan []byte),
		c:             nil,
		sdpChan:       sdpChan,
		sdpReplyChan:  sdpReplyChan,
		candidateChan: candidateChan,
	}
}

func (s *SignalingThread) SignalCandidate(candidate webrtc.ICECandidateInit) error {
	payload, err := json.Marshal(candidate)
	if err != nil {
		slog.Error("marshal error", "error", err)
		return err
	}
	s.c.WriteMessage(websocket.TextMessage, payload)
	slog.Info("send candidate", "candidate", string(payload))
	return nil
}

func (s *SignalingThread) Spin() {
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
			s.recv <- message
		}
	}()
	slog.Info("dial success")

	cfgMessage, err := json.Marshal(s.cfg.SessionConfig)
	if err != nil {
		slog.Error("compose message error", "error", err)
		return
	}
	wsConn.WriteMessage(websocket.TextMessage, cfgMessage)
	slog.Info("send configure message")
	recvRaw := <-s.recv
	sdp := webrtc.SessionDescription{}
	err = json.Unmarshal(recvRaw, &sdp)
	if err != nil {
		slog.Error("unmarshal error", "error", err)
		return
	}
	s.sdpChan <- sdp
	answer := <-s.sdpReplyChan // await answer from peer connection
	payload, err := json.Marshal(answer)
	if err != nil {
		slog.Error("marshal error", "error", err)
	}
	wsConn.WriteMessage(websocket.TextMessage, payload)
	slog.Info("send answer", "sdp", answer.SDP)
	for {
		candidateRaw := <-s.recv
		candidate := webrtc.ICECandidateInit{}
		err := json.Unmarshal(candidateRaw, &candidate)
		if err != nil {
			slog.Error("unmarshal error", "error", err)
			continue
		}
		slog.Info("recv candidate", "candidate", candidate)
		s.candidateChan <- candidate
	}
}
