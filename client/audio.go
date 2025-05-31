package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"time"

	"github.com/gordonklaus/portaudio"
)

const (
	sampleRate      = 48000
	framesPerBuffer = 960 // 20ms @ 48kHz mono
)

var (
	audioStream    *portaudio.Stream
	playbackStream *portaudio.Stream
	incomingAudio  = make(chan []int16, 100)
	serverConn     *net.UDPConn
)

func audioSend(samples []int16) {
	if serverConn == nil {
		fmt.Println("[SEND] Warning: serverConn is nil, cannot send")
		return
	}

	buf := make([]byte, 2+len(samples)*2)
	binary.LittleEndian.PutUint16(buf[0:2], 0x5541) // Prefix 'AU'
	binary.Write(sliceWriter(buf[2:]), binary.LittleEndian, samples)

	n, err := serverConn.Write(buf)
	if err != nil {
		fmt.Println("[SEND] Error sending audio packet:", err)
	} else {
		fmt.Printf("[SEND] Sent %d bytes to server %s\n", n, serverConn.RemoteAddr().String())
	}
}

func InitAudio() error {
	fmt.Println("[AUDIO] InitAudio() entered")

	// Set up input stream
	in := make([]int16, framesPerBuffer)
	inStream, err := portaudio.OpenDefaultStream(1, 0, sampleRate, len(in), in)
	if err != nil {
		return fmt.Errorf("failed to open input stream: %v", err)
	}
	audioStream = inStream

	// Set up output stream
	out := make([]int16, framesPerBuffer)
	outStream, err := portaudio.OpenDefaultStream(0, 1, sampleRate, len(out), &out)
	if err != nil {
		return fmt.Errorf("failed to open output stream: %v", err)
	}
	playbackStream = outStream

	// Start input stream
	if err := inStream.Start(); err != nil {
		return fmt.Errorf("failed to start input stream: %v", err)
	}
	fmt.Println("[AUDIO] Input stream started successfully")

	// Start output stream  
	if err := outStream.Start(); err != nil {
		return fmt.Errorf("failed to start output stream: %v", err)
	}
	fmt.Println("[AUDIO] Output stream started successfully")

	// Start input goroutine (DON'T close stream here)
	go func() {
		fmt.Println("[AUDIO] Input goroutine started")
		for {
			if IsPTTActive() {
				fmt.Println("[PTT] Held, attempting mic read")
				if err := inStream.Read(); err != nil {
					fmt.Println("[PTT] Mic read error:", err)
					continue
				}
				fmt.Printf("[PTT] Captured %d samples, max amplitude: %d\n", len(in), maxAmplitude(in))
				audioSend(in)
			} else {
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	// Start playback goroutine (DON'T close stream here)
	go func() {
		fmt.Println("[AUDIO] Playback goroutine started")
		for samples := range incomingAudio {
			fmt.Printf("[PLAYBACK] Playing %d samples, max amplitude: %d\n", len(samples), maxAmplitude(samples))
			copy(out, samples)
			if err := outStream.Write(); err != nil {
				fmt.Println("[AUDIO] Playback error:", err)
			} else {
				fmt.Println("[PLAYBACK] Successfully wrote to output stream")
			}
		}
	}()

	return nil
}

// Helper function to check if we're actually getting audio data
func maxAmplitude(samples []int16) int16 {
	var max int16 = 0
	for _, sample := range samples {
		if sample < 0 {
			sample = -sample
		}
		if sample > max {
			max = sample
		}
	}
	return max
}

func sliceWriter(buf []byte) *sliceBuffer {
	return &sliceBuffer{buf: buf}
}

type sliceBuffer struct {
	buf []byte
	off int
}

func (b *sliceBuffer) Write(p []byte) (int, error) {
	n := copy(b.buf[b.off:], p)
	b.off += n
	return n, nil
}

// TestAudioPipeline generates a test tone to verify audio playback works
func TestAudioPipeline() {
	fmt.Println("[TEST] Starting audio pipeline test...")
	
	// Generate a simple 440Hz sine wave (A note)
	testSamples := make([]int16, framesPerBuffer)
	for i := 0; i < framesPerBuffer; i++ {
		// Generate sine wave at 440Hz
		angle := 2.0 * 3.14159 * 440.0 * float64(i) / float64(sampleRate)
		amplitude := int16(8000 * math.Sin(angle)) // Moderate volume
		testSamples[i] = amplitude
	}
	
	fmt.Printf("[TEST] Generated %d test samples, max amplitude: %d\n", len(testSamples), maxAmplitude(testSamples))
	
	// Send to playback buffer
	select {
	case incomingAudio <- testSamples:
		fmt.Println("[TEST] Test audio queued for playback")
	default:
		fmt.Println("[TEST] Could not queue test audio - buffer full")
	}
}