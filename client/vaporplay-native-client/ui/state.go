package ui

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

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

func fetchGameData(configPath string) (*[]config.GameConfig, error) {
	cfg, err := clientconfig.LoadClientConfig(&configPath)
	if err != nil {
		return nil, err
	}
	resp, err := http.Get(cfg.Addr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	gameconfigs := &[]config.GameConfig{}
	err = json.Unmarshal(body, gameconfigs)
	if err != nil {
		return nil, err
	}
	return gameconfigs, nil

	// // mock
	// time.Sleep(1 * time.Second)

	// gameconfigs := &[]config.GameConfig{
	// 	config.GameConfig{
	// 		GameId:          "1",
	// 		GameDisplayName: "aaa",
	// 	},
	// 	config.GameConfig{
	// 		GameId:          "2",
	// 		GameDisplayName: "bbb",
	// 	},
	// }
	// return gameconfigs, nil
}
