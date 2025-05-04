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
func LoadClientConfig(configPath *string) (*ClientConfig, error) {
	if _, err := os.Stat(*configPath); errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	f, err := os.Open(*configPath)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	bf := make([]byte, stat.Size())
	_, err = bufio.NewReader(f).Read(bf)
	if err != nil && err != io.EOF {
		return nil, err
	}
	cfg := &ClientConfig{}
	err = json.Unmarshal(bf, cfg)
	if err != nil {
		return nil, err
	}
	err = f.Close()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func SaveClientConfig(configPath string, cfg *ClientConfig) error {
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		// create file
		f, err := os.Create(configPath)
		if err != nil {
			return err
		}
		b, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		n, err := f.Write(b)
		if err != nil || n != len(b) {
			return err
		}
		err = f.Close()
		if err != nil {
			return err
		}
	} else {
		// open file in write mode (clear original content)
		f, err := os.OpenFile(configPath, os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		b, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		n, err := f.Write(b)
		if err != nil || n != len(b) {
			return err
		}
		err = f.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
