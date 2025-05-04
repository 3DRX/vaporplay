package ui

import (
	"log/slog"

	clientconfig "github.com/3DRX/vaporplay/client/vaporplay-native-client/client-config"
	"github.com/3DRX/vaporplay/config"
)

func defaultCfg() *clientconfig.ClientConfig {
	return &clientconfig.ClientConfig{
		Addr: "10.129.89.200:8080",
		SessionConfig: config.SessionConfig{
			GameConfig: config.GameConfig{
				GameId:          "588650",
				GameWindowName:  "Dead Cells",
				GameDisplayName: "Dead Cells",
				GameIcon:        "",
				EndGameCommands: []config.KillProcessCommandConfig{
					config.KillProcessCommandConfig{
						ProcessName: "deadcells",
					},
				},
			},
			CodecConfig: config.CodecConfig{
				Codec:          "h264_nvenc",
				InitialBitrate: 10000000,
				FrameRate:      60,
				MaxBitrate:     20000000,
			},
		},
	}
}

func useClientConfig(configPath string) *clientconfig.ClientConfig {
	cfg, err := clientconfig.LoadClientConfig(&configPath)
	if err != nil {
		slog.Info("Load client config failed, using default config", "error", err)
		cfg := defaultCfg()
		clientconfig.SaveClientConfig(configPath, cfg)
		return cfg
	}
	return cfg
}

func setClientConfig(configPath string, handler func(*clientconfig.ClientConfig)) {
	cfg, err := clientconfig.LoadClientConfig(&configPath)
	if err != nil {
		slog.Info("Load client config failed, using default config", "error", err)
		cfg = defaultCfg()
	}
	handler(cfg)
	clientconfig.SaveClientConfig(configPath, cfg)
}
