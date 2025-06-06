# AHCLI - Self-Hosted Voice Chat That Just Works

> **Bulletproof VOIP with zero enshittification** 🎯

## What is AHCLI?

AHCLI is a **lightweight, self-hosted voice chat application** that brings back the simplicity of classic VOIP tools like Ventrilo, but with modern architecture and zero bullshit.

**Core Features:**
- **Crystal clear 48kHz audio** with <50ms latency
- **Push-to-talk system** with customizable bindings
- **Multi-channel support** with real-time user management
- **Self-hosted** - your server, your rules, your data
- **Zero tracking** - we don't know you exist and we like it that way
- **Terminal-style UI** with Kentucky cyberpunk aesthetics

## 🎯 Mission Statement

**A bulletproof, self-hosted VOIP tool that escapes modern enshittification.**

We reject what everything is becoming: paywalled features, tracking, forced accounts, microtransactions, data mining. AHCLI is the absolute opposite - a tool for people who still believe in owning what they run.

## 🎨 Design Philosophy

### Anti-Enshittification
- ✅ **Self-hosted**: Complete control, no external dependencies
- ✅ **Config-driven**: Plain JSON files, no complex wizards  
- ✅ **Zero tracking**: No spying, no data collection, no accounts
- ✅ **Zero subscriptions**: Pay once (nothing), use forever
- ✅ **Open source**: Fork it, mod it, make it yours

### Technical Excellence
- **Audio-first**: 48kHz sample rate, raw PCM, zero artifacts
- **Low latency**: <50ms end-to-end, optimized audio pipeline
- **Clean architecture**: Bulletproof core, hackable surface
- **Minimal design**: Everything serves a purpose, no bloat

## 🏗️ Architecture

### Backend (Go)
- **UDP audio streaming** for minimal latency
- **WebSocket state sync** for real-time UI updates
- **Embedded web files** for zero-dependency deployment
- **Premium audio processing** with noise gate and compression

### Frontend (Web)
- **Kentucky terminal styling** - dark green cyberpunk aesthetic
- **Modular components** - clean separation of concerns
- **Smart message routing** - user chat vs debug separation
- **Terminal-style interfaces** - functional and beautiful

### Current Structure
```
ahcli/
├── server/              # Go voice relay server
│   ├── server.exe       # Compiled binary
│   ├── config.json      # Server configuration
│   ├── main.go          # Server entry point
│   └── net.go           # UDP audio handling
├── client/              # Go client application
│   ├── client.exe       # Compiled binary
│   ├── settings.config  # Client configuration
│   ├── main.go          # Client entry point
│   ├── audio.go         # Audio processing pipeline
│   ├── webserver.go     # Embedded web UI server
│   └── web/             # Frontend assets
│       ├── index.html   # Main UI structure
│       ├── css/         # Kentucky terminal styling
│       ├── js/          # Modular JavaScript
│       └── components/  # UI components
├── common/              # Shared protocol definitions
└── build.bat           # Automated build system
```

## 🎧 Audio Quality Specs

- **Sample Rate**: 48kHz (crystal clear)
- **Frame Size**: 960 samples (20ms low latency)
- **Processing**: Premium noise gate, compression, makeup gain
- **Codec**: Raw PCM (no compression artifacts)
- **Latency**: <50ms end-to-end
- **Network**: Robust UDP with packet loss recovery

## 🚀 Quick Start

### For Users
1. **Run server**: `.\run-server.bat`
2. **Run client**: `.\run-client.bat`
3. **Start talking**: Hold LSHIFT to transmit

### For Developers
```bash
# Build everything
.\build.bat

# Run components separately
cd server && server.exe
cd client && client.exe
```

## ⚙️ Configuration

### Client Settings (`client/settings.config`)
```json
{
  "nickname": ["quikmn", "fallback1", "anon1337"],
  "preferred_server": "Home",
  "ptt_key": "LSHIFT",
  "audio_processing": {
    "noise_gate": {"enabled": true, "threshold_db": -40},
    "compressor": {"enabled": true, "threshold_db": -18, "ratio": 3.0},
    "makeup_gain": {"enabled": true, "gain_db": 6},
    "preset": "balanced"
  },
  "servers": {
    "Home": {"ip": "127.0.0.1:4422"}
  }
}
```

### Server Settings (`server/config.json`)
```json
{
  "server_name": "ahcli bunker",
  "listen_port": 4422,
  "motd": "Welcome to AHCLI - self-hosted voice chat.",
  "channels": [
    {"name": "General", "allow_speak": true},
    {"name": "AFK", "allow_speak": false}
  ]
}
```

## 🎮 User Interface

### Kentucky Terminal Aesthetic
- **Dark green terminal colors** - `#0a0e0a`, `#7c9f35`, `#c8e682`
- **Monospace typography** - authentic terminal feel
- **Clean functional layout** - three-column design
- **Smart message separation** - user chat vs system debug

### Core Features
- **User chat** - Terminal-style messaging in center panel
- **Debug terminal** - Professional diagnostic overlay (🔧 Debug button)
- **Audio controls** - Real-time visualization and processing controls
- **Channel management** - Click to switch, real-time user lists
- **System tray** - Minimize to tray, right-click menu

## 🔧 Development Principles

### Architecture First
- **Clean separation of concerns** - no mixed responsibilities
- **Minimal and purposeful** - everything serves a function
- **No hacks or band-aids** - if it goes in, it goes in right
- **Bulletproof core** - voice quality is untouchable

### Code Standards
- **Go backend** - performance and reliability
- **Modular frontend** - clean component architecture
- **Config-driven** - behavior defined in JSON files
- **Zero dependencies** - embedded assets, portable deployment

## 🎯 Current Status

### ✅ What's Working
- **Voice transmission & reception** - Crystal clear, zero crackling
- **Push-to-talk system** - Responsive, customizable keys
- **Multi-channel support** - Seamless switching
- **Premium audio processing** - Noise gate, compression, visualization
- **Modern web UI** - Kentucky terminal styling
- **Self-hosted deployment** - No external dependencies

### 🚧 Active Development
- **User chat system** - Terminal-style messaging between users
- **Debug terminal** - Professional diagnostic interface
- **Message routing** - Clean separation of chat vs system messages
- **Audio visualization** - Real-time processing feedback

### 🔮 Future Enhancements
- **E2E encryption** - Strong privacy protection
- **Voice activation (VOX)** - Alternative to push-to-talk
- **Audio compression** - OPUS codec for bandwidth efficiency
- **Mobile support** - Web-based cross-platform access

## 🎮 Supported PTT Keys
`LSHIFT`, `RSHIFT`, `LCTRL`, `RCTRL`, `SPACE`, `F1-F24`, `A-Z`, `0-9`, and more.

## 🏆 Performance Goals

- **Audio latency**: <50ms end-to-end
- **CPU usage**: Minimal impact on system
- **Memory footprint**: Lean and efficient
- **Network bandwidth**: Optimized for voice
- **Startup time**: Instant launch, no delays

## 📝 License

MIT License - Fork it, mod it, spread the love! 💕

This is open sauce for everyone to enjoy and do with what they please - as long as we get some love and fame.

---

**AHCLI: Self-hosted, simple, bulletproof voice chat that just works.** 🎧✨

*A tool for people who still believe in owning what they run.*