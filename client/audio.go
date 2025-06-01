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
		LogSend("Warning: serverConn is nil, cannot send")
		return
	}

	buf := make([]byte, 2+len(samples)*2)
	binary.LittleEndian.PutUint16(buf[0:2], 0x5541) // Prefix 'AU'
	binary.Write(sliceWriter(buf[2:]), binary.LittleEndian, samples)

	_, err := serverConn.Write(buf)
	if err != nil {
		LogSend("Error sending audio packet: %v", err)
		// Update TUI with error
		if !isTUIDisabled() {
			TUISetError("Audio send failed")
		}
	} else {
		// Update TUI with transmitted packet
		if !isTUIDisabled() {
			TUIIncrementTX()
		}
	}
}

func InitAudio() error {
	LogAudio("InitAudio() entered")

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
	LogAudio("Input stream started successfully")

	// Start output stream  
	if err := outStream.Start(); err != nil {
		return err
	}
	LogAudio("Output stream started successfully")

	// Start input goroutine (DON'T close stream here)
	go func() {
		LogAudio("Input goroutine started")
		var lastPTTState bool
		var frameCount int
		
		for {
			pttActive := IsPTTActive()
			
			// Update Console TUI with PTT state
			if !isTUIDisabled() {
				ConsoleTUISetPTT(pttActive)
			}
			
			// Only log PTT state changes, not every frame
			if pttActive != lastPTTState {
				if pttActive {
					LogPTT("Started transmitting")
					frameCount = 0 // Reset counter when starting transmission
					if !isTUIDisabled() {
						ConsoleTUIAddMessage("● Transmitting")
					}
				} else {
					LogPTT("Stopped transmitting")
					if !isTUIDisabled() {
						ConsoleTUIAddMessage("○ Ready")
					}
				}
				lastPTTState = pttActive
			}
			
			if pttActive {
				if err := inStream.Read(); err != nil {
					LogPTT("Mic read error: %v", err)
					continue
				}
				// Only log every 50 frames (once per second) if there's audio
				frameCount++
				maxAmp := maxAmplitude(in)
				
				// Update Console TUI with audio level
				if !isTUIDisabled() && maxAmp > 0 {
					level := int(float64(maxAmp) / 32767.0 * 100)
					ConsoleTUISetAudioLevel(level)
				}
				
				if maxAmp > 50 && frameCount%50 == 0 {
					LogPTT("Transmitting audio (amplitude: %d)", maxAmp)
				}
				audioSend(in)
			} else {
				// Reset audio level when not transmitting
				if !isTUIDisabled() {
					ConsoleTUISetAudioLevel(0)
				}
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	// Start playback goroutine (DON'T close stream here)
	go func() {
		LogAudio("Playback goroutine started")
		var playbackFrameCount int
		
		for samples := range incomingAudio {
			maxAmp := maxAmplitude(samples)
			// Only log every 50 frames (once per second) if there's meaningful audio
			playbackFrameCount++
			if maxAmp > 50 && playbackFrameCount%50 == 0 {
				LogPlayback("Playing audio (amplitude: %d)", maxAmp)
			}
			
			// Update Console TUI with received audio level
			if !isTUIDisabled() && maxAmp > 50 {
				level := int(float64(maxAmp) / 32767.0 * 100)
				ConsoleTUISetAudioLevel(level)
			}
			
			copy(out, samples)
			if err := outStream.Write(); err != nil {
				LogAudio("Playback error: %v", err)
				if !isTUIDisabled() {
					ConsoleTUIAddMessage("Audio playback failed")
				}
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
	LogTest("Starting audio pipeline test...")
	
	if !isTUIDisabled() {
		ConsoleTUIAddMessage("Playing test tone...")
	}
	
	// Generate a simple 440Hz sine wave (A note)
	testSamples := make([]int16, framesPerBuffer)
	for i := 0; i < framesPerBuffer; i++ {
		// Generate sine wave at 440Hz
		angle := 2.0 * 3.14159 * 440.0 * float64(i) / float64(sampleRate)
		amplitude := int16(8000 * math.Sin(angle)) // Moderate volume
		testSamples[i] = amplitude
	}
	
	LogTest("Generated %d test samples, max amplitude: %d", len(testSamples), maxAmplitude(testSamples))
	
	// Send to playback buffer
	select {
	case incomingAudio <- testSamples:
		LogTest("Test audio queued for playback")
		if !isTUIDisabled() {
			ConsoleTUIAddMessage("Test tone played successfully")
		}
	default:
		LogTest("Could not queue test audio - buffer full")
		if !isTUIDisabled() {
			ConsoleTUIAddMessage("Audio buffer full during test")
		}
	}
}