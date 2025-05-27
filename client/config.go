package main

import (
	"encoding/json"
	"os"
)

type ServerEntry struct {
	IP string `json:"ip"`
}

type ClientConfig struct {
	Nickname            []string              `json:"nickname"`
	PreferredServer     string                `json:"preferred_server"`
	PTTKey              string                `json:"ptt_key"`
	VoiceActivation     bool                  `json:"voice_activation_enabled"`
	ActivationThreshold float64               `json:"activation_threshold"`
	NoiseSuppression    bool                  `json:"noise_suppression"`
	Servers             map[string]ServerEntry `json:"servers"`
}

func loadClientConfig(path string) (*ClientConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config ClientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
