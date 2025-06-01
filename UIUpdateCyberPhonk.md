# UI Update "CyberPhunk" 🎯🔥

## **Phase A: Layout Architecture Overhaul**

### **Step 1: Custom Titlebar Implementation**
- **Remove browser decorations** (`--hide-title-bar` flag in browser launch)
- **Create custom titlebar** in HTML/CSS
- **Add draggable area** with `-webkit-app-region: drag`
- **Custom minimize/maximize/close** buttons with JavaScript handlers
- **Close button** → minimize to tray (don't kill app)
- **Kentucky Terminal styling** with smooth rounded corners

### **Step 2: Fixed Layout Structure** 
```
┌─────────────────────────────────────────┐
│  CUSTOM TITLEBAR (draggable)            │ ← New custom bar
├─────────────────────────────────────────┤
│  SECTION 3: Header (static, never scroll)│ ← Existing header
├─────┬───────────────────┬───────────────┤
│  S2 │       S1          │      S5       │
│Stats│  Terminal Chat    │   Channels    │ ← Scrollable columns
│(📊)│     (💬)          │   & Users     │
│     │                   │     (👥)      │
├─────┴───────────────────┴───────────────┤
│  SECTION 4: Footer (static, never scroll)│ ← Existing footer
└─────────────────────────────────────────┘
```

### **Step 3: Content Migration**
- **Move connection/network stats** → Section 2 (left sidebar)
- **Convert Section 1** → **Pure terminal-style chat** (NO CARDS!)
- **Add independent scrollbars** to Sections 1 & 2
- **Keep Section 5** exactly as-is (channels/users)

## **Phase B: Terminal Chat System Implementation**

### **Step 4: Terminal Chat Format**
```
[23:17:41] quikmn: Hey everyone, voice chat working great!
[23:17:45] user2: What's up? How's the audio quality?
[23:17:46] quikmn: Crystal clear 48kHz, no crackling at all
[23:17:48] admin: Server running smooth, 12ms latency
```

**Kentucky Terminal Colors:**
- **Timestamps** `[23:17:41]` → **#00BFFF** (bright cyan)
- **Usernames** `quikmn:` → **#00FF00** (bright green)  
- **Messages** `Hey everyone...` → **#E0E0E0** (light gray)
- **Own messages** → **#FFFF00** (yellow) for your own text

### **Step 5: Terminal Chat Styling**
```css
.terminal-chat {
  font-family: 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.4;
  background: #0C0C0C;
  color: #E0E0E0;
  padding: 10px;
  overflow-y: auto;
  white-space: pre-wrap;
}

.chat-line {
  margin: 1px 0;
}

.timestamp { color: #00BFFF; }
.username { color: #00FF00; }
.message { color: #E0E0E0; }
.own-message { color: #FFFF00; }
```

### **Step 6: Stats Panel (Section 2)**
- **Move current message log** from center to left sidebar
- **Connection info** (uptime, server, etc.)
- **Network stats** (RX/TX packets)
- **Audio status** (PTT key, levels)
- **Left-side scrollbar** for overflow
- **Terminal-style text formatting** for consistency

## **Phase C: Responsive Layout**

### **Step 7: CSS Grid Implementation**
```css
.app {
  display: grid;
  grid-template-rows: 30px 60px 1fr 50px; /* titlebar, header, body, footer */
  height: 100vh;
}

.main-content {
  display: grid;
  grid-template-columns: 200px 1fr 250px; /* stats, chat, channels */
  overflow: hidden; /* Prevent layout breaks */
}
```

### **Step 8: Scroll Containment**
- **Section 1**: `overflow-y: auto` with **terminal-style scrollbar**
- **Section 2**: `overflow-y: auto` with **terminal-style scrollbar**  
- **Section 3 & 4**: `position: sticky` - never scroll away
- **Section 5**: Keep existing behavior

## **Phase D: Integration & Testing**

### **Step 9: Browser Launch Updates**
```go
// Updated browser flags
browsers := [][]string{
    {"chrome", "--app=" + url, "--hide-title-bar", "--disable-web-security"},
    // ... other browsers with titlebar flags
}
```

### **Step 10: JavaScript Window Controls**
- **Minimize**: Call tray API to hide window
- **Maximize**: Toggle fullscreen state  
- **Close**: Minimize to tray (don't exit app)
- **Drag**: Make titlebar draggable area

## **Implementation Order (Safest → Most Complex)**

1. **🟢 Layout restructure** (CSS Grid, scrollable columns)
2. **🟢 Terminal chat styling** (replace cards with pure text)
3. **🟡 Content migration** (stats to sidebar, chat messages to terminal format)  
4. **🟡 Custom titlebar** (HTML/CSS/JS window controls)
5. **🔴 Browser integration** (remove decorations, test window behavior)

## **Success Criteria**

✅ **Fixed layout** - nothing can break the grid structure  
✅ **Independent scrolling** - each section scrolls without affecting others  
✅ **Terminal chat** - pure text with Kentucky color coding  
✅ **Custom titlebar** - draggable, functional, cyberpunk styled  
✅ **Stats sidebar** - connection info moved to left panel  
✅ **No layout breaks** - UI stays stable regardless of content  
✅ **Authentic terminal vibe** - monospace font, minimal styling, color-coded text

## **Kentucky Terminal Aesthetic**

- **Terminal chat**: Pure text, no backgrounds, monospace font
- **Color coding**: Cyan timestamps, green usernames, gray messages, yellow for own
- **Custom titlebar**: Dark (#0C0C0C) with green (#00FF00) accents
- **Stats panel**: Terminal-style info display
- **Consistent typography**: Courier New throughout

## **Prerequisites**

⚠️ **Complete Phase 2 & 3 of Separation of Concerns first!**
- Finish observer pattern implementation
- Remove direct WebTUI calls from core systems
- Ensure UI is pure observer of AppState

**This ensures UI redesign won't break voice chat functionality!**

---

**Status:** 📋 **PLANNED** - Ready to implement after separation of concerns is complete