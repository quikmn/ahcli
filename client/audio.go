// FILE: client/audio.go

package main

import (
	"encoding/binary"
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
		LogError("Warning: serverConn is nil, cannot send")
		return
	}

	buf := make([]byte, 2+len(samples)*2)
	binary.LittleEndian.PutUint16(buf[0:2], 0x5541) // Prefix 'AU'
	binary.Write(sliceWriter(buf[2:]), binary.LittleEndian, samples)

	_, err := serverConn.Write(buf)
	if err != nil {
		LogError("Error sending audio packet: %v", err)
		
		// PURE APPSTATE: Only update AppState - observer handles WebTUI
		appState.AddMessage("Audio send failed", "error")
	} else {
		// PURE APPSTATE: Only update AppState - observer handles WebTUI
		appState.IncrementTX()
	}
}

func InitAudio() error {
	LogInfo("InitAudio() entered")

	// Set up input stream
	in := make([]int16, framesPerBuffer)
	inStream, err := portaudio.OpenDefaultStream(1, 0, sampleRate, len(in), in)
	if err != nil {
		return err
	}
	audioStream = inStream

	// Set up output stream
	out := make([]int16, framesPerBuffer)
	outStream, err := portaudio.OpenDefaultStream(0, 1, sampleRate, len(out), &out)
	if err != nil {
		return err
	}
	playbackStream = outStream

	// Start input stream
	if err := inStream.Start(); err != nil {
		return err
	}
	LogInfo("Input stream started successfully")

	// Start output stream
	if err := outStream.Start(); err != nil {
		return err
	}
	LogInfo("Output stream started successfully")

	// Start input goroutine
	go func() {
		LogInfo("Input goroutine started")
		var lastPTTState bool
		var frameCount int

		for {
			pttActive := IsPTTActive()
			
			// PURE APPSTATE: Only update AppState - observer handles WebTUI
			appState.SetPTTActive(pttActive)

			// Only log PTT state changes, not every frame
			if pttActive != lastPTTState {
				if pttActive {
					LogInfo("Started transmitting")
					frameCount = 0 // Reset counter when starting transmission
					
					// PURE APPSTATE: Only update AppState - observer handles WebTUI
					appState.AddMessage("● Transmitting", "ptt")
				} else {
					LogInfo("Stopped transmitting")
					
					// PURE APPSTATE: Only update AppState - observer handles WebTUI
					appState.AddMessage("○ Ready", "info")
				}
				lastPTTState = pttActive
			}

			if pttActive {
				if err := inStream.Read(); err != nil {
					LogError("Mic read error: %v", err)
					continue
				}
				frameCount++
				maxAmp := maxAmplitude(in)

				// PURE APPSTATE: Only update AppState - observer handles WebTUI
				if maxAmp > 0 {
					level := int(float64(maxAmp) / 32767.0 * 100)
					appState.SetAudioLevel(level)
				}

				if maxAmp > 50 && frameCount%50 == 0 {
					LogInfo("Transmitting audio (amplitude: %d)", maxAmp)
				}
				audioSend(in)
			} else {
				// PURE APPSTATE: Reset audio level - observer handles WebTUI
				appState.SetAudioLevel(0)
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	// Start playback goroutine
	go func() {
		LogInfo("Playback goroutine started")
		var playbackFrameCount int

		for samples := range incomingAudio {
			maxAmp := maxAmplitude(samples)
			playbackFrameCount++
			if maxAmp > 50 && playbackFrameCount%50 == 0 {
				LogInfo("Playing audio (amplitude: %d)", maxAmp)
			}

			// PURE APPSTATE: Only update AppState - observer handles WebTUI
			if maxAmp > 50 {
				level := int(float64(maxAmp) / 32767.0 * 100)
				appState.SetAudioLevel(level)
			}

			copy(out, samples)
			if err := outStream.Write(); err != nil {
				LogError("Playback error: %v", err)
				
				// PURE APPSTATE: Only update AppState - observer handles WebTUI
				appState.AddMessage("Audio playback failed", "error")
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
	LogInfo("Starting audio pipeline test...")

	// PURE APPSTATE: Only update AppState - observer handles WebTUI
	appState.AddMessage("Playing test tone...", "info")

	// Generate a simple 440Hz sine wave (A note)
	testSamples := make([]int16, framesPerBuffer)
	for i := 0; i < framesPerBuffer; i++ {
		// Generate sine wave at 440Hz
		angle := 2.0 * 3.14159 * 440.0 * float64(i) / float64(sampleRate)
		amplitude := int16(8000 * math.Sin(angle)) // Moderate volume
		testSamples[i] = amplitude
	}

	LogInfo("Generated %d test samples, max amplitude: %d", len(testSamples), maxAmplitude(testSamples))

	// Send to playback buffer
	select {
	case incomingAudio <- testSamples:
		LogInfo("Test audio queued for playback")
		
		// PURE APPSTATE: Only update AppState - observer handles WebTUI
		appState.AddMessage("Test tone played successfully", "success")
	default:
		LogError("Could not queue test audio - buffer full")
		
		// PURE APPSTATE: Only update AppState - observer handles WebTUI
		appState.AddMessage("Audio buffer full during test", "error")
	}
}