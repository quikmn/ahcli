package main

import (
	"fmt"
	"time"

	"github.com/gordonklaus/portaudio"
)

const (
	sampleRate     = 48000
	framesPerBuffer = 960 // 20ms @ 48kHz, mono
)

var audioStream *portaudio.Stream

func InitAudio() error {
	err := portaudio.Initialize()
	if err != nil {
		return fmt.Errorf("portaudio init failed: %v", err)
	}

	in := make([]int16, framesPerBuffer)

	stream, err := portaudio.OpenDefaultStream(1, 0, sampleRate, len(in), in)
	if err != nil {
		return fmt.Errorf("failed to open audio stream: %v", err)
	}
	audioStream = stream

	go func() {
		defer stream.Close()

		if err := stream.Start(); err != nil {
			fmt.Println("Failed to start audio stream:", err)
			return
		}
		defer stream.Stop()

		fmt.Println("Audio input stream started.")

		for {
			if IsPTTActive() {
				if err := stream.Read(); err != nil {
					fmt.Println("Error reading from mic:", err)
					continue
				}
				// For now, just log it
				fmt.Printf("Captured %d samples (PTT active)\n", len(in))
				// TODO: Encode & send
			} else {
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	return nil
}

func ShutdownAudio() {
	if audioStream != nil {
		audioStream.Stop()
		audioStream.Close()
	}
	portaudio.Terminate()
}
