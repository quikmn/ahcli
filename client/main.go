package main

import (
	"fmt"
	"time"
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

	// Set PTT key from config
	pttKeyCode = keyNameToVKCode(config.PTTKey)
	if pttKeyCode == 0 {
		fmt.Println("Unsupported PTT key:", config.PTTKey)
		return
	}

	// Start PTT listener loop
	StartPTTListener()

	// Connect to server
	err = connectToServer(config)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// For debugging PTT status only
	go func() {
		for {
			if IsPTTActive() {
				fmt.Println("PTT held")
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// Keep the program running
	select {}
}
