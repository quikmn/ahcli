# AHCLI - Self-Hosted Voice Chat That Just Works

> **Bulletproof VOIP with zero enshittification** 🎯

## What is AHCLI?

AHCLI is a **lightweight, self-hosted voice chat application** with a modern web interface that brings back the simplicity of classic VOIP tools like Ventrilo, but with contemporary architecture and zero bullshit.

**Core Features:**
- **Crystal clear 48kHz audio** with <50ms latency and premium processing
- **Push-to-talk system** with customizable key bindings
- **Multi-channel support** with real-time user management and persistent chat
- **Modern web interface** with Kentucky cyberpunk terminal aesthetics
- **Self-hosted** - your server, your rules, your data
- **Zero tracking** - we don't know you exist and we like it that way
- **Embedded browser UI** - launches automatically in Chrome app mode

## 🎯 Mission Statement

**A bulletproof, self-hosted VOIP tool that escapes modern enshittification.**

We reject what everything is becoming: paywalled features, tracking, forced accounts, microtransactions, data mining. AHCLI is the absolute opposite - a tool for people who still believe in owning what they run.

## 🏗️ Modern Architecture

### Backend (Go)
- **UDP audio streaming** for minimal latency with premium processing pipeline
- **WebSocket state sync** for real-time UI updates
- **Embedded web server** serving the complete web interface
- **Premium audio processing** with noise gate, compression, and makeup gain
- **Transport encryption** with X25519 key exchange and ChaCha20-Poly1305
- **Persistent chat system** with channel-based routing and history

### Frontend (Web-based)
- **Kentucky terminal styling** - dark cyberpunk aesthetic with smooth animations
- **Modular component architecture** - clean separation of concerns
- **Audio processing controls** - noise gate, compression, and bypass controls
- **Multi-user chat system** - Terminal-style formatting with self-message styling
- **Smart message routing** - automatic separation of user chat vs system debug
- **Chrome app mode** - launches as native-feeling application

### Tech Stack
```
Frontend:    Vanilla JavaScript + CSS3 + HTML5
Backend:     Go + PortAudio + Gorilla WebSocket
Audio:       48kHz PCM with RNNoise + Custom Processing
Crypto:      X25519 + ChaCha20-Poly1305 (transport encryption)
Transport:   UDP (audio) + WebSocket (state) + HTTP (API)
UI:          Embedded web server → Chrome app mode
```

### Current Structure
```
ahcli/
├── server/              # Go voice relay server
│   ├── server.exe       # Compiled binary
│   ├── config.json      # Server configuration
│   ├── main.go          # Server entry point
│   ├── net.go           # UDP audio + WebSocket handling
│   ├── chat.go          # Persistent chat system
│   ├── crypto.go        # E2E encryption
│   └── state.go         # Client state management
├── client/              # Go client with embedded web UI
│   ├── client.exe       # Compiled binary
│   ├── settings.config  # Client configuration
│   ├── main.go          # Client entry point + tray integration
│   ├── audio.go         # Premium audio processing pipeline
│   ├── webserver.go     # Embedded web UI server
│   ├── appstate.go      # Centralized state management
│   ├── tray.go          # System tray integration
│   └── web/             # Complete web interface
│       ├── index.html   # Main UI structure
│       ├── css/         # Kentucky cyberpunk styling
│       ├── js/          # Modular JavaScript components
│       └── components/  # Reusable UI components
├── common/              # Shared libraries
│   ├── protocol.go      # Network protocol definitions
│   └── logger/          # Unified logging system
└── build.bat           # Automated build system
```

## 🎧 Audio Quality Specs

- **Sample Rate**: 48kHz (crystal clear, broadcast quality)
- **Frame Size**: 960 samples (20ms ultra-low latency)
- **Processing Chain**: Noise Gate → Dynamic Compressor → Makeup Gain
- **Codec**: Raw PCM (zero compression artifacts) with optional OPUS
- **Latency**: <50ms end-to-end with jitter buffering
- **Network**: Robust UDP with sequence tracking and loss recovery

## 🎨 User Interface

### Kentucky Terminal Aesthetic
- **Dark cyberpunk colors** - Deep purple-blue with mellow pink accents
- **Terminal typography** - Courier New with authentic monospace feel
- **Smooth animations** - Subtle transitions and visual feedback
- **Real-time visualization** - Professional audio processing meters

### Multi-User Chat System
- **Terminal-style formatting** - `[HH:MM] <username> message`
- **Self-message styling** - Your messages highlighted with orange accents
- **Channel persistence** - Chat history preserved per channel

### Launch Experience
- **Auto-launch** - Opens Chrome in app mode on startup
- **System tray** - Minimizes to tray, right-click menu
- **Native feel** - Borderless window, proper app icon
- **Instant access** - Left-click tray to open, always available

## 🚀 Quick Start

### For Users
1. **Run server**: `ahcli-server.exe`
2. **Run client**: `ahcli-client.exe`
3. **Start talking**: Hold LSHIFT to transmit (customizable)
4. **Web UI**: Opens automatically in browser

### For Developers
```bash
# Build everything
.\build.bat

# Run components separately
cd server && go run .
cd client && go run .
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
  ],
  "chat": {
    "enabled": true,
    "log_file": "chat.log",
    "max_messages": 100000,
    "load_recent_on_join": 100
  }
}
```

### Chat Features
- **Terminal-style formatting** - `[HH:MM] <username> message`
- **Self-message styling** - Your messages highlighted with orange accents
- **Channel persistence** - Chat history preserved per channel

## 🎮 Supported PTT Keys
`LSHIFT`, `RSHIFT`, `LCTRL`, `RCTRL`, `SPACE`, `F1-F24`, `A-Z`, `0-9`, and more.

## 🎯 Current Status

### ✅ What's Working
- **Voice transmission & reception** - Crystal clear audio
- **Push-to-talk system** - Customizable key bindings
- **Multi-channel support** - Channel switching with user lists
- **Web interface** - Kentucky cyberpunk UI
- **Multi-user chat** - Persistent, channel-based messaging
- **Transport encryption** - Secure chat with X25519 + ChaCha20-Poly1305
- **Self-hosted deployment** - No external dependencies

## 📝 License

MIT License - Fork it, mod it, spread the love! 💕

This is open sauce for everyone to enjoy and do with what they please - as long as we get some love and fame.

---

**AHCLI: Self-hosted, modern, bulletproof voice chat with a slick web interface.** 🎧✨

*A tool for people who still believe in owning what they run.*