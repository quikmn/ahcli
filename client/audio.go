package main

import (
	"encoding/json"
	"fmt"
	"time"
	"net"
	"github.com/gordonklaus/portaudio"
)

const (
	sampleRate     = 48000
	framesPerBuffer = 960 // 20ms @ 48kHz, mono
)

var (
	audioStream     *portaudio.Stream
	playbackStream  *portaudio.Stream
	incomingAudio   = make(chan []int16, 100)
	serverConn *net.UDPConn
)

func audioSend(data []byte) {
	if serverConn != nil {
		serverConn.Write(data)
	}
}

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

				packet := map[string]interface{}{
					"type": "audio",
					"data": in, // raw int16 slice
				}

				buf, err := json.Marshal(packet)
				if err != nil {
					fmt.Println("Failed to encode audio packet:", err)
					continue
				}

				audioSend(buf)
			} else {
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	return nil
}

func startPlayback() error {
	out := make([]int16, framesPerBuffer)

	var err error
	playbackStream, err = portaudio.OpenDefaultStream(0, 1, sampleRate, len(out), &out)
	if err != nil {
		return fmt.Errorf("failed to open playback stream: %v", err)
	}

	err = playbackStream.Start()
	if err != nil {
		return fmt.Errorf("failed to start playback stream: %v", err)
	}

	go func() {
		defer playbackStream.Stop()
		defer playbackStream.Close()

		for frame := range incomingAudio {
			if len(frame) != framesPerBuffer {
				continue
			}
			copy(out, frame)
			if err := playbackStream.Write(); err != nil {
				fmt.Println("Error writing to speaker:", err)
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
