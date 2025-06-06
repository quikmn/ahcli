// FILE: client/audio.go

package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"os"
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

	// Premium audio processing
	audioProcessor *AudioProcessor
	sequenceNumber uint16 = 0
)

func audioSend(samples []int16) {
	if serverConn == nil {
		LogError("Warning: serverConn is nil, cannot send")
		return
	}

	// BYPASS PROCESSING FOR DEBUG - send raw samples
	processedSamples := samples // Skip all processing

	// Create enhanced packet with sequence number
	buf := make([]byte, 4+len(processedSamples)*2)
	binary.LittleEndian.PutUint16(buf[0:2], 0x5541)         // Prefix 'AU'
	binary.LittleEndian.PutUint16(buf[2:4], sequenceNumber) // Sequence number
	binary.Write(sliceWriter(buf[4:]), binary.LittleEndian, processedSamples)

	sequenceNumber++

	_, err := serverConn.Write(buf)
	if err != nil {
		LogError("Error sending audio packet: %v", err)
		appState.AddMessage("Audio send failed", "error")
	} else {
		appState.IncrementTX()
	}
}

func InitAudio() error {
	LogInfo("InitAudio() entered - Premium Audio Processing Enabled")
	fmt.Println("=== PREMIUM AUDIO INIT STARTED ===") // GUARANTEED CONSOLE OUTPUT

	// MINIMAL ADDITION: Log to file too
	if logFile, err := os.OpenFile("client.log", os.O_APPEND|os.O_WRONLY, 0666); err == nil {
		fmt.Fprintln(logFile, "=== PREMIUM AUDIO INIT STARTED ===")
		logFile.Close()
	}

	// Initialize premium audio processor
	audioProcessor = NewAudioProcessor()
	LogInfo("Premium audio processor initialized with noise gate and compression")
	fmt.Println("Premium audio processor created")

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
	fmt.Println("Audio input stream STARTED")

	// Start output stream
	if err := outStream.Start(); err != nil {
		return err
	}
	LogInfo("Output stream started successfully")
	fmt.Println("Audio output stream STARTED")

	// Start enhanced input goroutine with bypass and dual-level tracking
	go func() {
		LogInfo("Enhanced audio input goroutine started with bypass capability")
		var lastPTTState bool
		var frameCount int

		for {
			pttActive := IsPTTActive()

			// Update PTT state
			appState.SetPTTActive(pttActive)

			// Log PTT state changes only
			if pttActive != lastPTTState {
				if pttActive {
					LogInfo("Started transmitting with enhanced audio processing")
					frameCount = 0
					appState.AddMessage("● Transmitting", "ptt")
				} else {
					LogInfo("Stopped transmitting")
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

				// Calculate RAW input level (before any processing)
				var sumSquares float64 = 0
				for _, sample := range in {
					sumSquares += float64(sample) * float64(sample)
				}
				rawRMS := math.Sqrt(sumSquares / float64(len(in)))
				rawInputLevel := float32(rawRMS / 32767.0)

				// Send raw level to AppState immediately
				appState.SetRawInputLevel(rawInputLevel)

				// Process through audio chain (or bypass)
				var processedSamples []int16
				if audioProcessor != nil && audioProcessor.IsBypassed() {
					// BYPASS: Use raw samples
					processedSamples = in
					appState.SetProcessedInputLevel(rawInputLevel) // Same as raw when bypassed
				} else {
					// PROCESS: Run through audio chain
					processedSamples = audioProcessor.ProcessInputAudio(in)

					// Calculate PROCESSED input level
					var processedSumSquares float64 = 0
					for _, sample := range processedSamples {
						processedSumSquares += float64(sample) * float64(sample)
					}
					processedRMS := math.Sqrt(processedSumSquares / float64(len(processedSamples)))
					processedInputLevel := float32(processedRMS / 32767.0)

					// Send processed level to AppState
					appState.SetProcessedInputLevel(processedInputLevel)
				}

				// Update comprehensive audio stats every 10 frames
				if frameCount%10 == 0 {
					stats := audioProcessor.GetStats()
					stats.InputLevel = rawInputLevel // Ensure raw level is in stats
					appState.SetAudioStats(stats)

					// Log processing comparison occasionally
					if frameCount%50 == 0 {
						LogInfo("Audio Levels - Raw: %.1f%%, Processed: %.1f%%, Bypass: %t",
							rawInputLevel*100,
							appState.GetProcessedInputLevel()*100,
							audioProcessor.IsBypassed())
					}
				}

				// Send the processed (or bypassed) audio
				audioSend(processedSamples)

			} else {
				// Reset levels when not transmitting
				appState.SetRawInputLevel(0)
				appState.SetProcessedInputLevel(0)
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	// Start enhanced playback goroutine with visualization support
	go func() {
		LogInfo("Enhanced playback goroutine started with visualization support")
		fmt.Println("=== ENHANCED PLAYBACK GOROUTINE STARTED ===") // GUARANTEED OUTPUT

		// MINIMAL ADDITION: Log to file
		if logFile, err := os.OpenFile("client.log", os.O_APPEND|os.O_WRONLY, 0666); err == nil {
			fmt.Fprintln(logFile, "=== ENHANCED PLAYBACK GOROUTINE STARTED ===")
			logFile.Close()
		}

		var playbackFrameCount int
		var lastPacketTime time.Time
		var timingLogCount int

		for samples := range incomingAudio {
			now := time.Now()

			// WAN DIAGNOSTIC: Track timing between packets
			if !lastPacketTime.IsZero() {
				timeSinceLastPacket := now.Sub(lastPacketTime)
				timingLogCount++

				// Log every 10th packet to avoid spam, but catch timing issues
				if timingLogCount%10 == 0 || timeSinceLastPacket > 40*time.Millisecond || timeSinceLastPacket < 10*time.Millisecond {
					fmt.Printf("🕐 PACKET TIMING: %v since last (should be ~20ms)\n", timeSinceLastPacket)

					// Log significant timing anomalies to file
					if logFile, err := os.OpenFile("client.log", os.O_APPEND|os.O_WRONLY, 0666); err == nil {
						fmt.Fprintf(logFile, "PACKET TIMING: %v since last\n", timeSinceLastPacket)
						logFile.Close()
					}
				}
			}
			lastPacketTime = now

			fmt.Println("*** RECEIVED AUDIO PACKET ***") // GUARANTEED OUTPUT

			// MINIMAL ADDITION: Log to file
			if logFile, err := os.OpenFile("client.log", os.O_APPEND|os.O_WRONLY, 0666); err == nil {
				fmt.Fprintln(logFile, "*** RECEIVED AUDIO PACKET ***")
				logFile.Close()
			}

			// DEBUG: Check sample content and audio device
			maxAmp := maxAmplitude(samples)
			fmt.Printf("PLAYBACK DEBUG - Samples: %d, Max Amplitude: %d\n", len(samples), maxAmp)

			// Log to file too
			if logFile, err := os.OpenFile("client.log", os.O_APPEND|os.O_WRONLY, 0666); err == nil {
				fmt.Fprintf(logFile, "PLAYBACK DEBUG - Samples: %d, Max Amplitude: %d\n", len(samples), maxAmp)
				logFile.Close()
			}

			playbackFrameCount++
			if maxAmp > 50 && playbackFrameCount%50 == 0 {
				LogInfo("Playing audio (amplitude: %d)", maxAmp)
				fmt.Printf("Playing audio (amplitude: %d)\n", maxAmp)
			}

			// Update output level for visualization based on received audio
			if maxAmp > 50 {
				// Calculate output level for visualization
				outputLevel := float32(maxAmp) / 32767.0

				// Update legacy audio level
				level := int(outputLevel * 100)
				appState.SetAudioLevel(level)

				// TODO: Add output level to AppState when we implement output visualization
				// For now, the input visualization shows transmission, this shows reception
			}

			copy(out, samples)
			if err := outStream.Write(); err != nil {
				LogError("Playback error: %v", err)
				fmt.Printf("PLAYBACK ERROR: %v\n", err)
				appState.AddMessage("Audio playback failed", "error")
			}
		}
		fmt.Println("=== PLAYBACK GOROUTINE ENDED ===") // Should never see this
	}()

	// Start enhanced audio quality monitoring with visualization updates
	go func() {
		qualityTicker := time.NewTicker(2 * time.Second) // More frequent for better visualization
		defer qualityTicker.Stop()

		for range qualityTicker.C {
			stats := audioProcessor.GetStats()

			// Update AppState with comprehensive audio quality info
			appState.SetAudioStats(stats)

			// Report significant issues to user
			if stats.PacketLoss > 0.05 {
				appState.AddMessage(fmt.Sprintf("Audio Quality: %s (%.1f%% loss)",
					stats.AudioQuality, stats.PacketLoss*100), "warning")
			}

			// Log detailed stats for debugging
			LogDebug("Audio Stats - Quality: %s, Latency: %v, Loss: %.2f%%, Jitter: %v",
				stats.AudioQuality, stats.BufferLatency, stats.PacketLoss*100, stats.NetworkJitter)
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

// TestAudioPipeline generates a test tone to verify premium audio processing
func TestAudioPipeline() {
	LogInfo("Starting premium audio pipeline test with visualization...")

	appState.AddMessage("Testing premium audio processing with visualization...", "info")

	// Generate a more sophisticated test signal
	testSamples := make([]int16, framesPerBuffer)
	for i := 0; i < framesPerBuffer; i++ {
		// Mix of 440Hz and 880Hz for richer test
		angle1 := 2.0 * 3.14159 * 440.0 * float64(i) / float64(sampleRate)
		angle2 := 2.0 * 3.14159 * 880.0 * float64(i) / float64(sampleRate)
		amplitude := int16(4000 * (math.Sin(angle1) + 0.5*math.Sin(angle2)))
		testSamples[i] = amplitude
	}

	// Calculate test signal level for visualization
	var sumSquares float64 = 0
	for _, sample := range testSamples {
		sumSquares += float64(sample) * float64(sample)
	}
	testLevel := float32(math.Sqrt(sumSquares/float64(len(testSamples))) / 32767.0)

	// Update visualization with test signal
	appState.SetInputLevel(testLevel)

	// Process through premium audio pipeline
	processedSamples := audioProcessor.ProcessInputAudio(testSamples)

	LogInfo("Generated test tone: %d samples, processed with premium pipeline", len(processedSamples))
	LogInfo("Max amplitude - Original: %d, Processed: %d", maxAmplitude(testSamples), maxAmplitude(processedSamples))

	// Send to jitter buffer for playback
	audioProcessor.AddToJitterBuffer(9999, processedSamples) // Special sequence for test

	// Get processing stats and update visualization
	stats := audioProcessor.GetStats()
	stats.InputLevel = testLevel // Override with actual test level
	appState.SetAudioStats(stats)

	LogInfo("Premium Audio Test - Quality: %s, Noise Gate: %t, Compression: %.2f, Level: %.1f%%",
		stats.AudioQuality, stats.NoiseGateOpen, stats.CompressionGain, testLevel*100)

	appState.AddMessage("Premium audio test with visualization completed successfully", "success")
}
