package config

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
)

type KillProcessCommandConfig struct {
	Flags       []string `json:"flags"`
	ProcessName string   `json:"process_name"`
}

type GameConfig struct {
	GameId          string                     `json:"game_id"`
	GameWindowName  string                     `json:"game_window_name"`
	GameDisplayName string                     `json:"game_display_name"`
	GameIcon        string                     `json:"game_icon"`
	EndGameCommands []KillProcessCommandConfig `json:"end_game_commands"`
}

type Config struct {
	Addr  string       `json:"addr"` // http service address
	Games []GameConfig `json:"games"`
}

func isValidAddr(addr *string) bool {
	// Try to separate hostname and port
	host, _, err := net.SplitHostPort(*addr)
	if err != nil {
		// If splitting fails, assume the entire string is a host
		host = *addr
	}

	// First try to parse as IP address
	ip := net.ParseIP(host)
	if ip != nil {
		// If it's a valid IP, check if it's IPv4
		return ip.To4() != nil
	}

	// Check if it's a valid hostname
	if isValidHostname(host) {
		return true
	}

	// If not a valid hostname, try to resolve it
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return false
	}

	// Check if any of the resolved IPs is IPv4
	for _, ip := range ips {
		if ip.To4() != nil {
			return true
		}
	}

	return false
}

func isValidHostname(host string) bool {
	// RFC 1123 hostname validation
	if len(host) > 255 {
		return false
	}
	// Host should not start or end with a dot
	if host[0] == '.' || host[len(host)-1] == '.' {
		return false
	}
	// Split hostname into labels
	labels := strings.Split(host, ".")
	// A valid hostname must have at least one label
	if len(labels) < 1 {
		return false
	}
	for _, label := range labels {
		if len(label) < 1 || len(label) > 63 {
			return false
		}
		// RFC 1123 allows hostname labels to start with a digit
		// Only allow alphanumeric characters and hyphens
		for i, c := range label {
			if !((c >= 'a' && c <= 'z') ||
				(c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9') ||
				(c == '-' && i > 0 && i < len(label)-1)) { // hyphen cannot be first or last
				return false
			}
		}
	}
	return true
}

func checkCfg(c *Config) error {
	if !isValidAddr(&c.Addr) {
		return fmt.Errorf("invalid ipv4 addr \"%s\"", c.Addr)
	}
	// TODO: check game configs
	return nil
}

func LoadCfg(cfgPath string) *Config {
	if _, err := os.Stat(cfgPath); errors.Is(err, os.ErrNotExist) {
		slog.Info(cfgPath + " not found, using default config")
		return &Config{
			Addr: "localhost:8080",
		}
	}
	f, err := os.Open(cfgPath)
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
	c := &Config{}
	err = json.Unmarshal(bf, c)
	if err != nil {
		panic(err)
	}
	err = checkCfg(c)
	if err != nil {
		panic(err)
	}

	// Print config
	slog.Info("config loaded", "config", c)
	return c
}
