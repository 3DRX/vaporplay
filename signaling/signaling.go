package signaling

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/3DRX/piongs/config"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

type SignalingThread struct {
	cfg                 *config.Config
	upgrader            *websocket.Upgrader
	conn                *websocket.Conn
	haveReceiverPromise chan struct{}
	sendSDPChan         <-chan webrtc.SessionDescription
	recvSDPChan         chan<- webrtc.SessionDescription
	sendCandidateChan   <-chan webrtc.ICECandidateInit
	recvCandidateChan   chan<- webrtc.ICECandidateInit
	connecting          bool
}

func NewSignalingThread(
	cfg *config.Config,
	sendSDPChan <-chan webrtc.SessionDescription,
	recvSDPChan chan<- webrtc.SessionDescription,
	sendCandidateChan <-chan webrtc.ICECandidateInit,
	recvCandidateChan chan<- webrtc.ICECandidateInit,
) *SignalingThread {
	return &SignalingThread{
		cfg: cfg,
		upgrader: &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		conn:                nil,
		haveReceiverPromise: make(chan struct{}),
		sendSDPChan:         sendSDPChan,
		recvSDPChan:         recvSDPChan,
		sendCandidateChan:   sendCandidateChan,
		recvCandidateChan:   recvCandidateChan,
		connecting:          false,
	}
}

func (s *SignalingThread) Spin() <-chan struct{} {
	mux := http.NewServeMux()
	mux.Handle("GET /webrtc", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.conn != nil {
			slog.Warn("already have a receiver, rejecting new connection")
			w.WriteHeader(http.StatusConflict)
			return
		}
		conn, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			panic(err)
		}
		slog.Info("new receiver connected")
		s.conn = conn
		go s.handleRecvMessages()
		go s.handleSendMessages()
	}))

	httpServer := &http.Server{
		Addr:    s.cfg.Addr,
		Handler: mux,
	}
	go func() {
		err := httpServer.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()
	return s.haveReceiverPromise
}

func (s *SignalingThread) handleSendMessages() {
	for {
		select {
		case sdp := <-s.sendSDPChan:
			jsonMsg, err := json.Marshal(sdp)
			if err != nil {
				slog.Error("failed to marshal SDP", "error", err)
			}
			err = s.conn.WriteMessage(websocket.TextMessage, jsonMsg)
			if err != nil {
				slog.Error("websocket write error", "error", err)
			}
			slog.Info("sent SDP", "sdp", sdp.SDP)
		case candidate := <-s.sendCandidateChan:
			jsonMsg, err := json.Marshal(candidate)
			if err != nil {
				slog.Error("failed to marshal ICE candidate", "error", err)
			}
			err = s.conn.WriteMessage(websocket.TextMessage, jsonMsg)
			if err != nil {
				slog.Error("websocket write error", "error", err)
			}
			slog.Info("sent ICE candidate", "candidate", candidate)
		}
	}
}

func (s *SignalingThread) handleRecvMessages() {
	for {
		_, message, err := s.conn.ReadMessage()
		if err != nil {
			slog.Error("websocket read error", "error", err)
			return
		}
		if string(message) != "Hello" {
			s.connecting = false
			continue
		}
		if !s.connecting {
			s.haveReceiverPromise <- struct{}{}
			_, message, err = s.conn.ReadMessage()
			if err != nil {
				slog.Error("websocket read error", "error", err)
				return
			}
			// try to parse it as an SDP
			newSDP := webrtc.SessionDescription{}
			err = json.Unmarshal(message, &newSDP)
			if err != nil {
				slog.Error("failed to parse message as SDP", "error", err)
				continue
			}
			slog.Info("received SDP", "sdp", newSDP.SDP)
			s.recvSDPChan <- newSDP
		}
	}
}
