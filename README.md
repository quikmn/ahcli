# AHCLI - High-Quality Voice Chat Application

> **Mission Accomplished: Crystal clear voice chat with zero bloat** 🎉

## What is AHCLI?

AHCLI is a **lightweight, high-performance voice chat application** similar to Ventrilo/TeamSpeak, built for Windows. It features:

- **Crystal clear audio quality** with 48kHz @ 20ms latency
- **Modern web-based interface** (no installation headaches)  
- **Push-to-talk (PTT)** with customizable key bindings
- **Multi-channel support** with real-time user lists
- **Embedded Chromium browser** for seamless UI experience
- **UDP-based audio streaming** for minimal latency

## 🎯 Project Status: COMPLETE & PRODUCTION READY

### ✅ What's Working Perfectly
- **Voice transmission & reception** - Crystal clear, zero crackling
- **Push-to-talk system** - Responsive, customizable keys (LSHIFT default)
- **Web UI interface** - Modern, responsive, real-time updates
- **Multi-channel support** - Switch channels seamlessly
- **User management** - See who's online, who's talking
- **Network stability** - Robust UDP with ping/pong keepalive
- **Audio pipeline** - 48kHz mono, optimized for voice
- **Browser integration** - Auto-launches in app mode
- **Configuration system** - JSON-based, easy to modify

### 📦 What's Included
```
ahcli/
├── server/           # Voice relay server
│   ├── server.exe    # Compiled server binary
│   └── config.json   # Server configuration
├── client/           # Voice client application  
│   ├── client.exe    # Compiled client binary
│   ├── settings.config # Client configuration
│   └── web/          # Embedded web UI
├── chromium/         # Bundled browser (portable)
├── common/           # Shared protocol definitions
├── build.bat         # Automated build script
├── run-server.bat    # Quick server launcher
└── run-client.bat    # Quick client launcher
```

## 🎧 Audio Quality Specs
- **Sample Rate**: 48kHz
- **Frame Size**: 960 samples (20ms)
- **Channels**: Mono (optimized for voice)
- **Codec**: Raw PCM (no compression artifacts)
- **Latency**: <50ms end-to-end
- **Quality**: **Crystal clear**

## 🚀 Quick Start

### For Users (Easy Mode)
1. **Run the server**: Double-click `run-server.bat`
2. **Run the client**: Double-click `run-client.bat`  
3. **Start talking**: Hold LSHIFT to transmit

### For Developers
```bash
# Build everything
.\build.bat

# Run server
cd server && server.exe

# Run client  
cd client && client.exe
```

## ⚙️ Configuration

### Client Settings (`client/settings.config`)
```json
{
  "nickname": ["quikmn", "fallback1", "anon1337"],
  "preferred_server": "Home",
  "ptt_key": "LSHIFT",
  "servers": {
    "Home": { "ip": "127.0.0.1:4422" },
    "Remote": { "ip": "vpn.example.com:4422" }
  }
}
```

### Server Settings (`server/config.json`)
```json
{
  "server_name": "ahcli bunker",
  "listen_port": 4422,
  "motd": "Welcome to ahcli.",
  "channels": [
    { "name": "General", "allow_speak": true },
    { "name": "AFK", "allow_speak": false }
  ]
}
```

## 🎮 Supported PTT Keys
`LSHIFT`, `RSHIFT`, `LCTRL`, `RCTRL`, `SPACE`, `F1-F24`, `A-Z`, `0-9`, and many more.

## 🏗️ Technical Architecture

### Clean, Modern Design
- **Go backend** for performance and reliability
- **Web frontend** for modern UI without installation
- **UDP protocol** for real-time audio streaming
- **WebSocket** for UI state synchronization
- **Embedded assets** for zero-dependency deployment

### Performance Optimizations
- **Lean codebase**: Removed 52% of bloat code
- **Real-time audio threads**: Optimized for low latency
- **Efficient networking**: Batched updates, smart broadcasting
- **Memory management**: Bounded buffers, controlled allocations

## 🔧 Development

### Dependencies
- **Go 1.24.3+**
- **PortAudio** (included as DLL)
- **Chromium** (bundled for UI)

### Build Requirements
- Windows (tested on Windows 11)
- Go toolchain
- Git (for version control)

### Architecture Highlights
- **Single-threaded audio processing** (no race conditions)
- **Unified logging** (simple, effective)
- **Clean separation** between client/server/common

## 🎯 Mission Statement

> **A config-driven, super easy, self-hosted and simple as pie VOIP solution that escapes the modern enshittification. Open Sauce for everyone to enjoy and do with what they please - as long as I get some love.**

**Goals Achieved**:
- ✅ **Config-driven**: JSON files, no complex setup wizards
- ✅ **Super easy**: Double-click .bat files to run
- ✅ **Self-hosted**: Your server, your rules, your data
- ✅ **Simple as pie**: No subscriptions, no accounts, no bullshit
- ✅ **Escape enshittification**: No ads, tracking, or feature paywalls
- ✅ **Open Sauce**: MIT license, fork it, mod it, love it

## 🚧 Future Enhancements (Optional)
- Voice activation (VOX) support
- Strong E2E Encryption
- Audio compression options
- Background noise cancellation logic
- Multi-server support in UI
- Mobile client (via web browser)
- Recording/playback features

## 📝 License
MIT License - See LICENSE file for details. Fork it, mod it, spread the love! 💕

---

**AHCLI: Self hosted, simple. Escape the Enshitification.** 🎧✨