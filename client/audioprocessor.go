// FILE: client/audioprocessor.go
package main

import (
	"ahcli/common/logger"
	"container/list"
	"sync"
	"time"
)

// AudioPacket represents a processed audio packet with metadata
type AudioPacket struct {
	SeqNum    uint16
	Timestamp uint32
	Data      []int16
	Size      int
	Received  time.Time
}

// NoiseGate removes background noise below a threshold
type NoiseGate struct {
	threshold   float32       // -40dB default
	attackTime  time.Duration // 2ms
	releaseTime time.Duration // 50ms
	holdTime    time.Duration // 100ms

	// State
	gateOpen   bool
	holdTimer  time.Time
	envelope   float32
	lastSample float32
}

// DynamicCompressor smooths out volume variations
type DynamicCompressor struct {
	threshold   float32       // -18dB
	ratio       float32       // 3:1 compression
	attackTime  time.Duration // 5ms
	releaseTime time.Duration // 100ms
	makeupGain  float32       // Auto-calculated

	// State
	envelope      float32
	gainReduction float32
}

// MakeupGain adds gain to compensate for compression
type MakeupGain struct {
	gainDB     float32 // Gain in decibels
	gainLinear float32 // Calculated linear gain
}

// JitterBuffer handles packet reordering and timing
type JitterBuffer struct {
	sync.RWMutex

	buffer     *list.List
	bufferTime time.Duration // 60ms default
	maxBuffer  time.Duration // 200ms max
	minBuffer  time.Duration // 20ms min

	// Adaptive parameters
	targetLatency time.Duration
	currentJitter time.Duration
	packetLoss    float32

	// Packet management
	expectedSeq   uint16
	lastTimestamp uint32
	packetsLost   int
	packetsTotal  int

	// Output timing
	nextPlayTime time.Time
	playInterval time.Duration // 20ms (960 samples @ 48kHz)
}

// AudioProcessor handles the complete audio processing chain
type AudioProcessor struct {
	// Input processing
	noiseGate  *NoiseGate
	compressor *DynamicCompressor
	makeupGain *MakeupGain

	// Network buffering
	jitterBuffer *JitterBuffer

	// Settings
	enableNoiseGate    bool
	enableCompressor   bool
	enableMakeupGain   bool
	enableJitterBuffer bool

	// NEW: Bypass functionality
	bypassProcessing bool

	// Statistics - INTERNAL ONLY (with mutex for thread safety)
	stats audioStatsInternal
}

// audioStatsInternal - internal stats with mutex (NOT exported)
type audioStatsInternal struct {
	sync.RWMutex

	// Input stats
	InputLevel      float32
	NoiseGateOpen   bool
	CompressionGain float32

	// Network stats
	NetworkJitter time.Duration

	// Quality metrics
	AudioQuality   string  // "Excellent", "Good", "Fair", "Poor"
	ProcessingLoad float32 // CPU usage estimate
}

// AudioStats - CLEAN export struct (NO MUTEX)
type AudioStats struct {
	// Input stats
	InputLevel      float32
	NoiseGateOpen   bool
	CompressionGain float32

	// Network stats
	BufferLatency time.Duration
	PacketLoss    float32
	NetworkJitter time.Duration

	// Quality metrics
	AudioQuality   string  // "Excellent", "Good", "Fair", "Poor"
	ProcessingLoad float32 // CPU usage estimate
}

// NewAudioProcessor creates a new audio processor with default settings
func NewAudioProcessor() *AudioProcessor {
	logger.Info("Creating new audio processor with premium settings")

	processor := &AudioProcessor{
		noiseGate: &NoiseGate{
			threshold:   -40.0, // dB
			attackTime:  2 * time.Millisecond,
			releaseTime: 50 * time.Millisecond,
			holdTime:    100 * time.Millisecond,
			envelope:    0.0,
		},
		compressor: &DynamicCompressor{
			threshold:   -18.0, // dB
			ratio:       3.0,   // 3:1 compression
			attackTime:  5 * time.Millisecond,
			releaseTime: 100 * time.Millisecond,
			makeupGain:  1.2, // Compensate for compression
			envelope:    0.0,
		},
		makeupGain: &MakeupGain{
			gainDB:     6.0, // +6dB default
			gainLinear: 2.0, // Calculated from gainDB
		},
		jitterBuffer: &JitterBuffer{
			buffer:        list.New(),
			bufferTime:    60 * time.Millisecond,
			maxBuffer:     200 * time.Millisecond,
			minBuffer:     20 * time.Millisecond,
			targetLatency: 80 * time.Millisecond,
			playInterval:  20 * time.Millisecond, // 960 samples @ 48kHz
		},
		enableNoiseGate:    true,  // Was false
		enableCompressor:   true,  // Was false
		enableMakeupGain:   true,  // Was false
		enableJitterBuffer: false, // TEMPORARILY DISABLED FOR DEBUGGING

		// NEW: Initialize bypass to false
		bypassProcessing: false,
	}

	logger.Debug("Audio processor initialized - NoiseGate: %t, Compressor: %t, MakeupGain: %t, JitterBuffer: %t",
		processor.enableNoiseGate, processor.enableCompressor, processor.enableMakeupGain, processor.enableJitterBuffer)

	return processor
}

// ProcessInputAudio processes audio from microphone before transmission
func (ap *AudioProcessor) ProcessInputAudio(samples []int16) []int16 {
	if len(samples) == 0 {
		logger.Debug("Empty audio samples received, returning as-is")
		return samples
	}

	processed := make([]int16, len(samples))
	copy(processed, samples)

	// Stage 1: Noise Gate
	if ap.enableNoiseGate {
		processed = ap.applyNoiseGate(processed)
	}

	// Stage 2: Dynamic Compressor
	if ap.enableCompressor {
		processed = ap.applyCompressor(processed)
	}

	// Stage 3: Makeup Gain
	if ap.enableMakeupGain {
		processed = ap.applyMakeupGain(processed)
	}

	// Update input statistics
	ap.updateInputStats(samples, processed)

	return processed
}

// applyMakeupGain applies makeup gain to compensate for compression
func (ap *AudioProcessor) applyMakeupGain(samples []int16) []int16 {
	mg := ap.makeupGain
	processed := make([]int16, len(samples))

	// Convert dB to linear gain if needed
	if mg.gainLinear == 0 {
		mg.gainLinear = powf(10.0, mg.gainDB/20.0)
		logger.Debug("Calculated linear gain: %.2f from %.1fdB", mg.gainLinear, mg.gainDB)
	}

	for i, sample := range samples {
		// Apply linear gain
		gained := float32(sample) * mg.gainLinear

		// Soft clipping to prevent harsh distortion
		if gained > 32767 {
			gained = 32767
		} else if gained < -32767 {
			gained = -32767
		}

		processed[i] = int16(gained)
	}

	return processed
}

// AddToJitterBuffer adds a received packet to the jitter buffer
func (ap *AudioProcessor) AddToJitterBuffer(seqNum uint16, data []int16) {
	if !ap.enableJitterBuffer {
		// Direct playback if jitter buffer disabled
		logger.Debug("Jitter buffer disabled, skipping packet %d", seqNum)
		return
	}

	packet := &AudioPacket{
		SeqNum:    seqNum,
		Timestamp: uint32(time.Now().UnixNano() / 1000000), // milliseconds
		Data:      data,
		Size:      len(data),
		Received:  time.Now(),
	}

	logger.Debug("Adding packet %d to jitter buffer (%d samples)", seqNum, len(data))
	ap.jitterBuffer.addPacket(packet)
}

// GetNextAudioFrame retrieves the next audio frame from jitter buffer
func (ap *AudioProcessor) GetNextAudioFrame() []int16 {
	if !ap.enableJitterBuffer {
		return nil
	}

	return ap.jitterBuffer.getNextFrame()
}

// applyNoiseGate applies noise gate processing to audio samples
func (ap *AudioProcessor) applyNoiseGate(samples []int16) []int16 {
	ng := ap.noiseGate
	processed := make([]int16, len(samples))

	for i, sample := range samples {
		// Convert to float for processing
		floatSample := float32(sample) / 32767.0

		// Calculate envelope (RMS-like)
		ng.envelope = ng.envelope*0.99 + floatSample*floatSample*0.01

		// Threshold in linear scale (approximate)
		thresholdLinear := powf(10.0, ng.threshold/20.0)

		// Gate logic
		if ng.envelope > thresholdLinear {
			if !ng.gateOpen {
				ng.gateOpen = true
				ng.holdTimer = time.Now().Add(ng.holdTime)
				logger.Debug("Noise gate opened (envelope: %.4f)", ng.envelope)
			}
		} else {
			if ng.gateOpen && time.Now().After(ng.holdTimer) {
				ng.gateOpen = false
				logger.Debug("Noise gate closed (envelope: %.4f)", ng.envelope)
			}
		}

		// Apply gate
		if ng.gateOpen {
			processed[i] = sample
		} else {
			processed[i] = 0 // Silence when gate closed
		}
	}

	// Update stats
	ap.stats.Lock()
	ap.stats.NoiseGateOpen = ng.gateOpen
	ap.stats.Unlock()

	return processed
}

// applyCompressor applies dynamic compression to audio samples
func (ap *AudioProcessor) applyCompressor(samples []int16) []int16 {
	comp := ap.compressor
	processed := make([]int16, len(samples))

	for i, sample := range samples {
		// Convert to float for processing
		floatSample := float32(sample) / 32767.0

		// Calculate level (envelope following)
		level := absf(floatSample)

		// Smooth envelope
		if level > comp.envelope {
			comp.envelope = level // Fast attack
		} else {
			comp.envelope = comp.envelope*0.999 + level*0.001 // Slow release
		}

		// Compression calculation
		thresholdLinear := powf(10.0, comp.threshold/20.0)
		if comp.envelope > thresholdLinear {
			// Above threshold: apply compression
			excess := comp.envelope - thresholdLinear
			reduction := excess * (1.0 - 1.0/comp.ratio)
			comp.gainReduction = 1.0 - reduction
		} else {
			// Below threshold: no compression
			comp.gainReduction = 1.0
		}

		// Apply compression and makeup gain
		compressedSample := floatSample * comp.gainReduction * comp.makeupGain

		// Soft limiting to prevent clipping
		if compressedSample > 1.0 {
			compressedSample = 1.0
		} else if compressedSample < -1.0 {
			compressedSample = -1.0
		}

		// Convert back to int16
		processed[i] = int16(compressedSample * 32767.0)
	}

	// Update stats
	ap.stats.Lock()
	ap.stats.CompressionGain = comp.gainReduction
	ap.stats.Unlock()

	return processed
}

// addPacket adds a packet to the jitter buffer in the correct order
func (jb *JitterBuffer) addPacket(packet *AudioPacket) {
	jb.Lock()
	defer jb.Unlock()

	jb.packetsTotal++

	// Check for packet loss
	if jb.expectedSeq != 0 && packet.SeqNum != jb.expectedSeq {
		if packet.SeqNum > jb.expectedSeq {
			// Packets were lost
			lost := int(packet.SeqNum - jb.expectedSeq)
			jb.packetsLost += lost
			logger.Debug("Packet loss detected: expected %d, got %d (%d packets lost)",
				jb.expectedSeq, packet.SeqNum, lost)
		} else {
			// Out-of-order packet (late arrival)
			logger.Debug("Out-of-order packet: expected %d, got %d (late arrival)",
				jb.expectedSeq, packet.SeqNum)
		}
	}

	jb.expectedSeq = packet.SeqNum + 1

	// Insert packet in sequence order
	inserted := false
	for e := jb.buffer.Front(); e != nil; e = e.Next() {
		existing := e.Value.(*AudioPacket)
		if packet.SeqNum < existing.SeqNum {
			jb.buffer.InsertBefore(packet, e)
			inserted = true
			break
		}
	}

	if !inserted {
		jb.buffer.PushBack(packet)
	}

	// Adaptive buffer sizing based on jitter
	jb.adaptBufferSize()

	// Remove old packets (prevent buffer overflow)
	maxPackets := int(jb.bufferTime / jb.playInterval)
	for jb.buffer.Len() > maxPackets {
		removed := jb.buffer.Remove(jb.buffer.Front())
		removedPacket := removed.(*AudioPacket)
		logger.Debug("Removed old packet %d from jitter buffer (overflow prevention)", removedPacket.SeqNum)
	}

	logger.Debug("Jitter buffer now contains %d packets (target: %d)", jb.buffer.Len(), maxPackets)
}

// getNextFrame retrieves the next audio frame when it's time to play
func (jb *JitterBuffer) getNextFrame() []int16 {
	jb.Lock()
	defer jb.Unlock()

	now := time.Now()

	// Initialize play timing - FIXED
	if jb.nextPlayTime.IsZero() {
		if jb.buffer.Len() > 0 {
			// Start playing immediately when we have packets
			jb.nextPlayTime = now
			logger.Info("Jitter buffer initialized - starting playback immediately with %d packets", jb.buffer.Len())
		} else {
			// No packets yet, wait
			return nil
		}
	}

	// Check if it's time to play next frame - SIMPLIFIED
	if now.Before(jb.nextPlayTime) {
		return nil // Not time yet
	}

	// Update next play time
	jb.nextPlayTime = jb.nextPlayTime.Add(jb.playInterval)

	// Get next packet from buffer
	if jb.buffer.Len() == 0 {
		// Buffer underrun - return silence and log it
		logger.Debug("Jitter buffer underrun - returning silence")
		return make([]int16, framesPerBuffer)
	}

	// Remove and return first packet
	element := jb.buffer.Front()
	jb.buffer.Remove(element)
	packet := element.Value.(*AudioPacket)

	logger.Debug("Jitter buffer: playing packet %d with %d samples", packet.SeqNum, len(packet.Data))
	return packet.Data
}

// adaptBufferSize adjusts buffer size based on network conditions
func (jb *JitterBuffer) adaptBufferSize() {
	oldBufferTime := jb.bufferTime

	// Calculate current packet loss rate
	if jb.packetsTotal > 0 {
		jb.packetLoss = float32(jb.packetsLost) / float32(jb.packetsTotal)
	}

	// Increase buffer size if packet loss is high
	if jb.packetLoss > 0.05 { // 5% packet loss
		jb.bufferTime = minDuration(jb.bufferTime+10*time.Millisecond, jb.maxBuffer)
	} else if jb.packetLoss < 0.01 { // Less than 1% packet loss
		jb.bufferTime = maxDuration(jb.bufferTime-5*time.Millisecond, jb.minBuffer)
	}

	// Log buffer size changes
	if jb.bufferTime != oldBufferTime {
		logger.Debug("Jitter buffer size adapted: %.1fms -> %.1fms (packet loss: %.2f%%)",
			float64(oldBufferTime)/float64(time.Millisecond),
			float64(jb.bufferTime)/float64(time.Millisecond),
			jb.packetLoss*100)
	}
}

// updateInputStats updates audio processing statistics
func (ap *AudioProcessor) updateInputStats(original, processed []int16) {
	// Calculate input level (RMS)
	var sum float64
	for _, sample := range original {
		val := float64(sample) / 32767.0
		sum += val * val
	}
	rms := powf(float32(sum/float64(len(original))), 0.5)

	ap.stats.Lock()
	ap.stats.InputLevel = rms

	// Update audio quality assessment
	if ap.jitterBuffer.packetLoss < 0.01 && ap.stats.NetworkJitter < 30*time.Millisecond {
		ap.stats.AudioQuality = "Excellent"
	} else if ap.jitterBuffer.packetLoss < 0.05 && ap.stats.NetworkJitter < 60*time.Millisecond {
		ap.stats.AudioQuality = "Good"
	} else if ap.jitterBuffer.packetLoss < 0.10 {
		ap.stats.AudioQuality = "Fair"
	} else {
		ap.stats.AudioQuality = "Poor"
	}

	ap.stats.Unlock()
}

// GetStats returns current audio processing statistics - FIXED (no mutex copy)
func (ap *AudioProcessor) GetStats() AudioStats {
	ap.stats.RLock()
	defer ap.stats.RUnlock()

	// Return CLEAN copy without any mutex - FIXED
	return AudioStats{
		InputLevel:      ap.stats.InputLevel,
		NoiseGateOpen:   ap.stats.NoiseGateOpen,
		CompressionGain: ap.stats.CompressionGain,
		BufferLatency:   ap.jitterBuffer.bufferTime,
		PacketLoss:      ap.jitterBuffer.packetLoss,
		NetworkJitter:   ap.stats.NetworkJitter,
		AudioQuality:    ap.stats.AudioQuality,
		ProcessingLoad:  ap.stats.ProcessingLoad,
	}
}

// Helper functions
func powf(base, exp float32) float32 {
	if exp == 0 {
		return 1
	}
	if exp == 0.5 {
		return sqrtf(base)
	}
	// Simple approximation for common cases
	result := float32(1)
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

func sqrtf(x float32) float32 {
	// Newton's method approximation
	if x <= 0 {
		return 0
	}
	guess := x / 2
	for i := 0; i < 10; i++ {
		guess = (guess + x/guess) / 2
	}
	return guess
}

func absf(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

// SetBypass enables or disables audio processing bypass
func (ap *AudioProcessor) SetBypass(bypass bool) {
	ap.stats.Lock()
	defer ap.stats.Unlock()

	oldBypass := ap.bypassProcessing
	ap.bypassProcessing = bypass

	if oldBypass != bypass {
		logger.Info("Audio processing bypass changed: %t -> %t", oldBypass, bypass)
	}
}

// IsBypassed returns whether audio processing is currently bypassed
func (ap *AudioProcessor) IsBypassed() bool {
	ap.stats.RLock()
	defer ap.stats.RUnlock()
	return ap.bypassProcessing
}

// GetBypassState returns current bypass state (thread-safe)
func (ap *AudioProcessor) GetBypassState() bool {
	ap.stats.RLock()
	defer ap.stats.RUnlock()
	return ap.bypassProcessing
}
