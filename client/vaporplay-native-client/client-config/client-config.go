package clientconfig

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/3DRX/vaporplay/config"
)

type ClientConfig struct {
	SessionConfig config.SessionConfig `json:"session_config"`
	Addr          string               `json:"addr"`
}

// load client config from configPath
func LoadClientConfig(configPath *string) *ClientConfig {
	if _, err := os.Stat(*configPath); errors.Is(err, os.ErrNotExist) {
		panic(*configPath + " not found, using default config")
	}
	f, err := os.Open(*configPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		panic(err)
	}
	bf := make([]byte, stat.Size())
	_, err = bufio.NewReader(f).Read(bf)
	if err != nil && err != io.EOF {
		panic(err)
	}
	cfg := &ClientConfig{}
	err = json.Unmarshal(bf, cfg)
	if err != nil {
		panic(err)
	}
	return cfg
}
