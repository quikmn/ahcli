# 🎯 AHCLI Channel-Bound Chat Implementation Plan

## Context for Next Claude
**AHCLI** is a self-hosted voice chat application following **"bulletproof, self-hosted VOIP that escapes enshittification"** philosophy. We've successfully implemented:

✅ **Voice transmission** - 48kHz, <50ms latency, working perfectly  
✅ **Channel switching** - users can switch between channels, UI updates properly  
✅ **Message routing** - clean separation of user chat vs debug messages  
✅ **Debug export** - simple log export (removed complex debug terminal)  
✅ **Premium audio processing** - bypassed for voice stability but keeps network features  
✅ **Beautiful cyberpunk UI** - mellow terminal aesthetic with proper colors  

## Current Chat Status
**Chat exists but is CLIENT-SIDE ONLY**:
- Messages typed in chat input go through `MessageRouter.sendChatMessage()`
- Messages are displayed locally but **NOT sent to other users**
- No server-side chat handling yet
- UI shows compact WoW-style chat: `[05:16] <quikmn> message`

## Mission: Implement Channel-Bound Chat
Create **real multi-user chat** where users in same voice channel can chat with each other.

## Architecture Requirements

### Core Principles (FOLLOW THESE):
- **"If it goes in, it goes in right"** - complete solution, no band-aids
- **Clean separation of concerns** - each component has one responsibility  
- **Minimal and purposeful** - everything serves voice chat coordination
- **Self-hosted** - no external dependencies, one log file approach
- **Anti-enshittification** - simple, owned by users

### File Structure Context:
```
server/
├── main.go           # Server entry point
├── net.go           # Network handling (has channel switching working)
├── state.go         # Client state management  
├── config.json      # Server configuration
└── logger.go        # Logging utilities

client/
├── main.go          # Client entry point
├── net.go           # Network handling (receives channel updates)
├── appstate.go      # Application state management
└── web/js/
    ├── message-router.js  # Routes messages (user chat vs debug)
    ├── user-chat.js       # Chat UI (WoW-style formatting)
    └── app.js            # Main controller
```

## Implementation Plan

### STEP 1: Server-Side Chat Storage (Simple Log Approach)

**Add to `server/config.json`:**
```json
{
  "server_name": "ahcli bunker",
  "listen_port": 4422,
  "motd": "Welcome to ahcli.",
  "channels": [
    { 
      "guid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "name": "General", 
      "allow_speak": true 
    },
    { 
      "guid": "f5e6d7c8-e5f6-7890-abcd-ef1234567890",
      "name": "AFK", 
      "allow_speak": false 
    }
  ],
  "chat": {
    "enabled": true,
    "log_file": "chat.log", 
    "max_messages": 100000,
    "load_recent_on_join": 100
  }
}
```

**Create chat storage in `server/` (new file or extend existing):**
- **In-memory storage**: `map[string][]ChatMessage` (channel GUID -> messages)
- **Log file**: Append-only `chat.log` with format: `2025-06-03T05:25:30Z [guid:a1b2c3d4] [General] <quikmn> message`
- **Circular buffer**: When 100k messages reached, drop oldest 10k
- **Load on startup**: Read last 100 messages per channel GUID from log
- **GUID system**: Channels have permanent GUIDs + changeable names for stable history

### STEP 2: Server-Side Chat Protocol

**Add to `server/net.go` in the JSON message switch:**
```go
case "chat":
    var chatMsg struct {
        Type     string `json:"type"`
        Channel  string `json:"channel"`   // Channel GUID for routing
        Message  string `json:"message"`
        Username string `json:"username"`
    }
    // Validate, store by GUID, and broadcast to channel users
```

**Broadcast chat messages to users in same channel only (matched by GUID).**

### STEP 3: Client-Side Chat Integration

**Update `client/net.go` to handle incoming chat:**
```go
case "chat_message":
    // Receive chat from other users
    // Route through MessageRouter to UserChat display
```

**Update `client/web/js/user-chat.js`:**
- `sendMessage()` should send to server instead of local-only
- Handle incoming messages from other users
- Maintain current WoW-style formatting: `[HH:MM] <username> message`

### STEP 4: Channel History on Join

When user joins a channel:
1. **Server sends recent messages** (last 100) to new user
2. **Client displays chat history** for context
3. **New messages** appear in real-time

## Technical Details

### Message Flow:
```
User types → UserChat.sendMessage() → Server chat handler → 
Broadcast to channel users → Other clients receive → Display in UserChat
```

### Chat Message Format:
**Server storage:** `2025-06-03T05:25:30Z [guid:a1b2c3d4] [General] <quikmn> hey everyone`  
**Client display:** `[05:25] <quikmn> hey everyone`

### GUID + Name System:
- **Permanent GUID**: Each channel gets UUID on creation - never changes
- **Changeable name**: UI-friendly names can be updated without losing history  
- **Log format**: `[guid:uuid] [current-name]` for both stability and readability
- **Filtering power**: `grep "guid:a1b2c3d4" chat.log` shows all history even after renames

### Key Functions Needed:
- `generateChannelGUID()` - create UUIDs for new channels
- `storeChatMessage(channelGUID, username, message)` - server storage by GUID
- `broadcastChatToChannel(channelGUID, chatMessage)` - server broadcast by GUID  
- `loadRecentMessages(channelGUID, count)` - server history by GUID
- `handleIncomingChat(chatData)` - client processing

## Existing Code Integration Points

### Already Working:
- **Channel switching** - users properly move between channels
- **User lists** - server tracks who is in what channel
- **Message routing** - client separates user chat from debug messages
- **WebSocket communication** - real-time updates working

### Extend These Files:
- **`server/net.go`** - add chat message handling in existing switch statement (route by GUID)
- **`client/net.go`** - add chat message handling in existing switch statement  
- **`client/web/js/user-chat.js`** - change from local-only to server-bound
- **`client/web/js/message-router.js`** - route incoming server chat correctly
- **`server/state.go`** - extend channel structure to include GUID field
- **`server/main.go`** - generate GUIDs for existing channels on first startup

## Expected Behavior After Implementation

1. **User joins #General** → sees last 100 chat messages for context (filtered by GUID)
2. **User types message** → appears for all users currently in #General  
3. **User switches to #AFK** → sees different chat history for #AFK channel (different GUID)
4. **Admin renames "General" to "Main"** → all chat history preserved, future logs show new name
5. **Server restart** → chat history preserved (loaded from chat.log, indexed by GUID)
6. **Multiple users** → can chat with each other in real-time per channel
7. **Grep filtering** → `grep "guid:a1b2c3d4" chat.log` shows all history regardless of renames

## File Size Reality Check
- **100k messages** ≈ 6.6MB (66 bytes per message average)
- **Less than a single photo** in storage
- **Months/years** of chat history
- **Set it and forget it** - no rotation complexity needed

## How Next Claude Should Behave

### Development Approach:
- **Follow AHCLI core philosophy** (see `corephilosophy.md`)
- **Clean, complete implementations** - no hacks or band-aids
- **Test voice chat after each change** - protect core functionality
- **Specify exact file paths** and be explicit about changes
- **Remove old code completely** before building new systems

### Code Communication:
- Always specify: `// FILE: path/to/file.go`  
- Be explicit: "Replace entire function" vs "Add to existing"
- Give complete sections, not surgical insertions
- User should never hunt for what you mean

### Implementation Priority:
1. **Server-side storage and broadcasting** (core functionality)
2. **Client-side integration** (connect to server chat)  
3. **Channel history on join** (user experience)
4. **Log file persistence** (data durability)

### Testing Approach:
After each step, ensure:
- **Voice transmission still works** (never break core functionality)
- **Channel switching still works** (don't regress existing features)
- **Chat appears in correct format** (maintain WoW-style UI)
- **Messages are channel-isolated** (no cross-channel leaking)

## Success Criteria

✅ **Real multi-user chat** - users can chat with each other  
✅ **Channel-bound** - chat isolated per voice channel  
✅ **Chat history** - context when joining channels  
✅ **Log persistence** - survives server restarts  
✅ **WoW-style formatting** - compact, clean display  
✅ **Voice unaffected** - chat doesn't break voice transmission  

## Final Notes

This feature transforms AHCLI from **voice-only** to **voice + coordination chat** - exactly like classic TeamSpeak/Ventrilo but self-hosted and enshittification-free.

The implementation should be **simple, robust, and purposeful** - focused on enhancing voice communication, not replacing it.

**Remember: This is a tool that works, not a product that sells.** 🎧