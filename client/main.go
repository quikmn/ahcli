package main

import (
	"encoding/json"
	"fmt"
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

func main() {
	config, err := loadClientConfig("settings.config")
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	fmt.Println("Client config loaded:")
	fmt.Printf("Preferred Nicknames: %v\n", config.Nickname)
	fmt.Printf("Preferred Server: %s\n", config.PreferredServer)
	fmt.Printf("PTT Key: %s\n", config.PTTKey)
	fmt.Printf("Voice Activation: %v\n", config.VoiceActivation)
	fmt.Printf("Threshold: %f\n", config.ActivationThreshold)
	fmt.Printf("Noise Suppression: %v\n", config.NoiseSuppression)
	fmt.Println("Servers:")
	for name, entry := range config.Servers {
		fmt.Printf(" - %s -> %s\n", name, entry.IP)
	}

	err = connectToServer(config)
	if err != nil {
		fmt.Println("Error:", err)
	}
}
