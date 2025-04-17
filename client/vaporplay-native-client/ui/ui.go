package ui

import (
	"log/slog"
	"time"

	clientconfig "github.com/3DRX/vaporplay/client/vaporplay-native-client/client-config"
	"github.com/3DRX/vaporplay/config"
)

type UIThread struct {
	startGamePromise chan *clientconfig.ClientConfig
}

func NewUIThread() *UIThread {
	return &UIThread{
		startGamePromise: make(chan *clientconfig.ClientConfig),
	}
}

func (u *UIThread) Spin() <-chan *clientconfig.ClientConfig {
	go u.doSomeThing()
	return u.startGamePromise
}

func (u *UIThread) doSomeThing() {
	// wait for 2 seconds, mock user input
	time.Sleep(2 * time.Second)

	cfg := &clientconfig.ClientConfig{
		Addr: "localhost:8080",
		SessionConfig: config.SessionConfig{
			GameConfig: config.GameConfig{
				GameId:          "383870",
				GameWindowName:  "Firewatch",
				GameDisplayName: "Fire Watch",
				GameIcon:        "",
				EndGameCommands: []config.KillProcessCommandConfig{{
					Flags:       []string{},
					ProcessName: "fw.x86_64",
				}},
			},
			CodecConfig: config.CodecConfig{
				Codec:          "h264_nvenc",
				InitialBitrate: 5_000_000,
				MaxBitrate:     20_000_000,
				FrameRate:      60,
			},
		},
	}
	slog.Info("start game", "game_id", cfg.SessionConfig.GameConfig.GameId)
	u.startGamePromise <- cfg
}
