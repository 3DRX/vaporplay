package clientconfig

import "github.com/3DRX/vaporplay/config"

type ClientConfig struct {
	SessionConfig config.SessionConfig
	Addr           string
}
