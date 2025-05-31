// FILE: client/main.go

package main

import (
	"fmt"
	"time"

	"github.com/gordonklaus/portaudio"
)

func main() {
	// Initialize PortAudio globally
	err := portaudio.Initialize()
	if err != nil {
		fmt.Println("PortAudio init failed:", err)
		return
	}
	defer portaudio.Terminate()

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

	StartPTTListener()

	// *** CRITICAL FIX: Initialize audio system ***
	fmt.Println("[MAIN] Initializing audio...")
	err = InitAudio()
	if err != nil {
		fmt.Println("[MAIN] Audio initialization failed:", err)
		return
	}
	fmt.Println("[MAIN] Audio initialized successfully")

	// Test audio pipeline
	go func() {
		time.Sleep(3 * time.Second) // Wait for everything to initialize
		TestAudioPipeline()
	}()

	// Start connection loop
	err = connectToServer(config)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Debug: monitor PTT state
	go func() {
		for {
			if IsPTTActive() {
				fmt.Println("[PTT] Debug: key held")
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	select {}
}