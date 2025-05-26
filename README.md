# ahcli

**Terminal-native, encrypted VOIP with zero bullshit.**

`ahcli` is a self-hosted, minimal voice chat system designed for power users. It features push-to-talk, voice activation, noise suppression, and secure communication — all configurable from plain-text files and operated entirely from the command line.

---

## 🔥 Features

- 🎙 Push-to-talk and voice activation support  
- 🔐 AES-GCM encrypted voice packets with shared key  
- 🧼 RNNoise-powered background noise suppression  
- 🧩 Channel-based architecture (with AFK zones)  
- 🧠 Terminal UI with real-time user and channel info  
- ⚙️ Configuration-only setup (no accounts, no GUI)  
- 🧳 Multi-server support via hash-map config  
- 🧍 Unique nickname assignment with fallbacks  
- 👑 Remote admin elevation via admin key  

---

## 📁 Folder Structure

```
ahcli/
├── client/           # CLI client with audio + TUI
├── server/           # Self-hosted VOIP server
├── common/           # Shared protocol/crypto utilities
├── README.md
├── .gitignore
```

---

## ⚙️ Configuration

### `client/settings.config`

```json
{
  "nickname": ["quikmn", "fallback1", "fallback2"],
  "preferred_server": "Home",
  "ptt_key": "T",
  "voice_activation_enabled": true,
  "activation_threshold": 0.015,
  "noise_suppression": true,
  "servers": {
    "Home": { "ip": "192.168.1.10:4422" },
    "Remote": { "ip": "vpn.example.com:4422" }
  }
}
```

### `server/config.json`

```json
{
  "server_name": "ahcli bunker",
  "listen_port": 4422,
  "shared_key": "hex-shared-key",
  "admin_key": "secret-admin-token",
  "motd": "Welcome to ahcli.",
  "channels": [
    { "name": "General", "allow_speak": true },
    { "name": "AFK", "allow_speak": false, "allow_listen": false }
  ]
}
```

---

## 🚀 How It Works

- Server enforces nickname uniqueness and tracks channel state  
- Clients use config to connect, transmit audio, and view state  
- Audio is passed through RNNoise → Opus → AES-GCM → UDP  
- Server relays voice to users in the same channel only  
- Admins can be elevated via `/elevate <key>`

---

## 📦 Status

🛠 In development — V1 planning complete.  
💬 Want to contribute? PRs and forks welcome once scaffolding is public.

---

## 🪪 License

MIT — do whatever you want, just don’t blame us when your bunker goes silent.
