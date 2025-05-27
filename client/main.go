package main

import (
	"fmt"
)

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
