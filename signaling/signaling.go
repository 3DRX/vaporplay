package signaling

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/3DRX/piongs/config"
	"github.com/3DRX/piongs/middleware"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

type SignalingThread struct {
	cfg                 *config.Config
	upgrader            *websocket.Upgrader
	conn                *websocket.Conn
	haveReceiverPromise chan *config.SessionConfig
	sendSDPChan         <-chan webrtc.SessionDescription
	recvSDPChan         chan<- webrtc.SessionDescription
	sendCandidateChan   <-chan webrtc.ICECandidateInit
	recvCandidateChan   chan<- webrtc.ICECandidateInit
	connecting          bool
	httpServer          *http.Server
	webuiDir            http.FileSystem
}

func NewSignalingThread(
	cfg *config.Config,
	sendSDPChan <-chan webrtc.SessionDescription,
	recvSDPChan chan<- webrtc.SessionDescription,
	sendCandidateChan <-chan webrtc.ICECandidateInit,
	recvCandidateChan chan<- webrtc.ICECandidateInit,
	webuiDir http.FileSystem,
) *SignalingThread {
	return &SignalingThread{
		cfg: cfg,
		upgrader: &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		conn:                nil,
		haveReceiverPromise: make(chan *config.SessionConfig),
		sendSDPChan:         sendSDPChan,
		recvSDPChan:         recvSDPChan,
		sendCandidateChan:   sendCandidateChan,
		recvCandidateChan:   recvCandidateChan,
		connecting:          false,
		webuiDir:            webuiDir,
	}
}

func (s *SignalingThread) Spin() <-chan *config.SessionConfig {
	fileServer := http.FileServer(s.webuiDir)
	mux := http.NewServeMux()
	mux.Handle("/", fileServer)
	mux.Handle("GET /games", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonGames, err := json.Marshal(s.cfg.Games)
		if err != nil {
			slog.Error("failed to marshal games", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(jsonGames)
		return
	}))
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
		Addr: s.cfg.Addr,
		Handler: middleware.ChainMiddleware(
			mux,
			middleware.CORSMiddleware,
		),
	}
	s.httpServer = httpServer
	go func() {
		err := httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()
	return s.haveReceiverPromise
}

func (s *SignalingThread) Close() error {
	if s.httpServer != nil {
		if s.conn != nil {
			if err := s.conn.Close(); err != nil {
				return err
			}
		}
		return s.httpServer.Shutdown(context.Background())
	}
	return nil
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
		selectedGame := &config.SessionConfig{}
		err = json.Unmarshal(message, selectedGame)
		if err != nil {
			s.connecting = false
			continue
		}
		if !s.connecting {
			s.haveReceiverPromise <- selectedGame
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
