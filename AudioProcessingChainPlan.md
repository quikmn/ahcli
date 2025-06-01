# Premium Audio Processing Chain Design 🎧💎

## **Architecture Overview**

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   MICROPHONE    │ →  │  INPUT PIPELINE  │ →  │    NETWORK     │
│    (Raw PCM)    │    │   (Processing)   │    │  (Compressed)  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                ↓
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│    SPEAKERS     │ ←  │ OUTPUT PIPELINE  │ ←  │    NETWORK     │
│   (Smooth PCM)  │    │   (Buffering)    │    │  (Compressed)  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## **Input Pipeline (Transmission Chain)**

### **Stage 1: Audio Capture & Conditioning**
```go
type AudioProcessor struct {
    // Input conditioning
    noiseGate      *NoiseGate
    compressor     *DynamicCompressor
    normalizer     *VolumeNormalizer
    
    // Encoding
    opusEncoder    *OpusEncoder
    
    // Quality settings
    bitrate        int    // 32-64 kbps adaptive
    complexity     int    // 0-10 (CPU vs quality)
}
```

**Processing Steps:**
1. **Noise Gate** - Eliminate background hum, keyboard clicks
2. **Dynamic Compressor** - Smooth out volume variations  
3. **Volume Normalizer** - Consistent output levels
4. **OPUS Encoding** - Professional compression

### **Stage 2: Smart Noise Gate**
```go
type NoiseGate struct {
    threshold      float32  // -40dB default
    attackTime     time.Duration  // 2ms
    releaseTime    time.Duration  // 50ms
    holdTime       time.Duration  // 100ms
    
    // Adaptive learning
    backgroundNoise float32
    adaptiveThreshold bool
}
```

**Features:**
- **Automatic threshold adjustment** based on ambient noise
- **Fast attack, smooth release** - no cutting off words
- **Hold time** prevents choppy speech during pauses

### **Stage 3: Dynamic Compressor**
```go
type DynamicCompressor struct {
    threshold      float32   // -18dB 
    ratio          float32   // 3:1 gentle compression
    attackTime     time.Duration  // 5ms
    releaseTime    time.Duration  // 100ms
    makeupGain     float32   // Auto-calculated
    
    // Advanced features
    lookAhead      time.Duration  // 5ms preview
    sideChain      bool          // Future: ducking support
}
```

**Benefits:**
- **Prevents audio clipping** from loud voices
- **Brings up quiet speech** for consistency  
- **Broadcast-quality dynamics** control
- **Transparent operation** - you don't notice it working

### **Stage 4: OPUS Encoding**
```go
type OpusEncoder struct {
    bitrate        int     // 32000-64000 bps
    sampleRate     int     // 48000 Hz
    channels       int     // 1 (mono)
    frameSize      int     // 960 samples (20ms)
    
    // Quality settings
    complexity     int     // 8 (good CPU/quality balance)
    vbr            bool    // Variable bitrate
    application    int     // OPUS_APPLICATION_VOIP
}
```

**Adaptive Bitrate Logic:**
- **Quiet/silence:** 16 kbps (minimal bandwidth)
- **Normal speech:** 32 kbps (crystal clear)  
- **Loud/music:** 64 kbps (high fidelity)
- **Network congestion:** Drop to 24 kbps gracefully

## **Output Pipeline (Reception Chain)**

### **Stage 1: Jitter Buffer & Packet Management**
```go
type JitterBuffer struct {
    buffer         []AudioPacket
    bufferTime     time.Duration  // 60ms default
    maxBufferTime  time.Duration  // 200ms max
    minBufferTime  time.Duration  // 20ms min
    
    // Adaptive sizing
    networkJitter  time.Duration  // Measured jitter
    packetLoss     float32        // Loss percentage
    adaptiveMode   bool           // Auto-adjust buffer size
    
    // Packet recovery
    lastSeqNum     uint16
    missingPackets map[uint16]time.Time
    interpolation  bool           // Fill gaps with audio
}
```

**Jitter Buffer Logic:**
- **Start with 60ms buffer** - good balance of latency/smoothness
- **Measure network jitter** - adapt buffer size dynamically
- **Packet loss detection** - identify missing sequences
- **Audio interpolation** - fill small gaps to prevent clicks

### **Stage 2: OPUS Decoding & Recovery**
```go
type OpusDecoder struct {
    decoder        *opus.Decoder
    
    // Error recovery
    errorConcealment bool    // OPUS built-in recovery
    lastGoodFrame    []int16 // For interpolation
    fadeInOut        bool    // Smooth transitions
}
```

**Recovery Features:**
- **OPUS error concealment** - built-in packet loss recovery
- **Frame interpolation** - smooth over missing packets
- **Fade in/out** - prevent audio pops from packet loss

### **Stage 3: Audio Smoothing & Output**
```go
type AudioSmoother struct {
    // Smoothing
    crossfadeTime  time.Duration  // 5ms crossfade
    volumeSmooth   *VolumeFilter  // Prevent sudden jumps
    
    // Output conditioning  
    limiter        *SoftLimiter   // Final safety net
    dcRemoval      *DCFilter      // Remove DC offset
    
    // Quality enhancement
    antialias      *AntiAliasFilter
    warmth         *WarmthFilter   // Subtle analog-style warmth
}
```

## **Network Protocol Enhancements**

### **Enhanced Packet Structure**
```go
type AudioPacket struct {
    Header struct {
        Magic      uint16  // 0x5541 'AU'
        SeqNum     uint16  // Sequence number
        Timestamp  uint32  // RTP-style timestamp
        Bitrate    uint16  // Current bitrate (adaptive)
        Flags      uint8   // DTX, FEC flags
    }
    
    Payload []byte  // OPUS compressed data
    FEC     []byte  // Forward Error Correction (optional)
}
```

### **Adaptive Quality System**
```go
type QualityManager struct {
    targetLatency  time.Duration  // 80ms target
    packetLoss     float32        // Current loss rate
    networkRTT     time.Duration  // Round trip time
    
    // Auto-adjustment
    bitrateTarget  int            // Current target bitrate
    bufferTarget   time.Duration  // Current buffer target
    
    // Quality ladder
    qualityLevels []QualityLevel
}

type QualityLevel struct {
    Bitrate     int           // 16, 24, 32, 48, 64 kbps
    BufferTime  time.Duration // Corresponding buffer time
    Complexity  int           // OPUS complexity setting
}
```

## **Configuration & User Control**

### **Audio Quality Presets**
```json
{
  "profiles": {
    "economy": {
      "bitrate": 24,
      "buffer_ms": 40,
      "processing": "light"
    },
    "balanced": {
      "bitrate": 32,
      "buffer_ms": 60,
      "processing": "standard"
    },
    "premium": {
      "bitrate": 64,
      "buffer_ms": 80,
      "processing": "full"
    },
    "music": {
      "bitrate": 128,
      "buffer_ms": 100,
      "processing": "full",
      "stereo": true
    }
  }
}
```

### **Auto-Detection Features**
- **Network quality assessment** - measure jitter/loss automatically
- **CPU usage monitoring** - scale processing based on available resources
- **Audio device capabilities** - adjust based on hardware sample rates
- **Background noise profiling** - adaptive noise gate training

## **Implementation Priority**

### **Phase 1: Foundation** 🟢
1. **OPUS integration** - replace raw PCM with OPUS compression
2. **Basic jitter buffer** - fixed 60ms buffer with packet reordering
3. **Simple noise gate** - fixed threshold background noise removal

### **Phase 2: Intelligence** 🟡  
1. **Adaptive jitter buffer** - dynamic sizing based on network conditions
2. **Dynamic compressor** - smooth volume variations
3. **Packet loss recovery** - interpolation and error concealment

### **Phase 3: Premium Features** 🔴
1. **Adaptive bitrate** - quality scaling based on network conditions
2. **Advanced audio processing** - limiter, warmth, anti-aliasing
3. **Quality presets** - user-selectable profiles for different use cases

## **Expected Results**

### **Audio Quality Improvements**
✅ **Crystal clear speech** - professional broadcast quality  
✅ **Consistent volume** - no more shouting/whispering  
✅ **Network resilient** - graceful degradation over poor connections  
✅ **Efficient bandwidth** - 50% less data than current raw PCM  

### **User Experience**
✅ **"It just works"** - automatic quality adjustment  
✅ **Lower latency** - smarter buffering reduces perceived delay  
✅ **Fewer dropouts** - packet loss recovery keeps conversation flowing  
✅ **Professional feel** - sounds better than Discord/TeamSpeak  

### **Technical Benefits**
✅ **Separation maintained** - all processing in dedicated audio module  
✅ **Modular design** - each component can be improved independently  
✅ **Future-proof** - foundation for advanced features like spatial audio  
✅ **Resource efficient** - OPUS is lighter on CPU than raw PCM processing  

## **Kentucky Terminal Integration**

All audio processing will be **completely transparent** to the UI layer:
- **AppState** gets simple audio level updates
- **No UI dependencies** in audio processing chain  
- **Clean status reporting** - "Audio Quality: Premium (64kbps)"
- **Settings integration** - quality profiles selectable in UI

**The voice chat becomes bulletproof while maintaining clean architecture!** 🎯💚