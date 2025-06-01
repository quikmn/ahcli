# Separation of Concerns Refactor Plan

## Objective

Decouple core functionality (audio, network) from UI presentation to enable safe UI experimentation without risk of breaking voice chat functionality.

## Current Problem

**Tight Coupling Example:**
```go
// Audio processing doing UI work - BAD
go func() {
    for {
        pttActive := IsPTTActive()
        WebTUISetPTT(pttActive)                    // Audio → UI coupling
        WebTUIAddMessage("● Transmitting", "ptt")  // Audio → UI coupling
        audioSend(samples)  // Actual audio work
    }
}()
```

**Issues:**
- Core systems directly call UI functions
- Can't change UI without touching audio/network code
- Risk of breaking voice chat while styling buttons
- Mixed concerns in single functions

## Target Architecture

**Clean Separation:**
```go
// Core systems - pure functionality
appState.SetPTTActive(pttActive)  // Just set state
audioSend(samples)                // Just do audio work

// UI layer - pure presentation  
appState.OnPTTChange(func(active bool) {
    updatePTTIndicator(active)     // UI decides how to present
    addMessage("Transmitting")     // UI decides messaging
})
```

**Benefits:**
- Core functionality = bulletproof black box
- UI = experimental playground
- Zero dependencies from core → UI
- Independent testing possible

## Implementation Plan

### Phase 1: Create State Bridge (No Breakage)

#### Step 1: Add State Manager
Create `client/appstate.go`:

```go
type AppState struct {
    // Audio state
    PTTActive     bool
    AudioLevel    int
    
    // Network state  
    Connected     bool
    CurrentChannel string
    Users         []string
    
    // Observer pattern
    observers     []func(StateChange)
}
```

#### Step 2: Dual-Write Pattern
Modify existing code to update BOTH WebTUI AND new state:

```go
// In audio.go - ADD alongside existing calls
pttActive := IsPTTActive()
appState.SetPTTActive(pttActive)  // NEW
WebTUISetPTT(pttActive)          // OLD - keep working
```

**Safety**: Everything still works exactly the same.

### Phase 2: Observer Pattern (Still No Breakage)

#### Step 3: Create UI Observer
Make WebTUI system also listen to state changes:

```go
// Observer calls existing WebTUI functions
appState.OnPTTChange(func(active bool) {
    WebTUISetPTT(active)  // Same function, different trigger
})
```

#### Step 4: Test Dual System
Verify:
- Core systems write to both old WebTUI AND new state
- New state also triggers WebTUI updates  
- Everything works twice but safely

### Phase 3: Cut the Cord (Gradual)

#### Step 5: Remove Direct WebTUI Calls
Go file by file, removing direct WebTUI calls:

```go
// OLD - Remove this
WebTUISetPTT(pttActive)
WebTUIAddMessage("Transmitting", "ptt")

// NEW - Replace with this
appState.SetPTTActive(pttActive)  // Observer handles UI
```

#### Step 6: Verify After Each File
Test voice chat after each file change. If broken, revert and debug.

### Phase 4: Pure Separation

#### Step 7: Core Systems = State Only
Final audio.go:
```go
// Pure audio processing
for {
    pttActive := IsPTTActive()
    appState.SetPTTActive(pttActive)  // Just state
    
    if pttActive {
        audioSend(samples)  // Just audio
    }
}
```

#### Step 8: UI = Pure Observer
All UI updates through state observation:
```go
// Pure presentation logic
appState.OnPTTChange(func(active bool) {
    updatePTTIndicator(active)
    addMessage(active ? "Transmitting" : "Ready")
})
```

## Refactoring Patterns Used

- **Extract State Pattern**: Centralize scattered state
- **Observer Pattern**: UI watches state changes
- **Dependency Inversion**: Core depends on state interface, not UI
- **Separation of Concerns**: Audio does audio, UI does UI

## Safety Measures

### Rollback Strategy
- Small git commits (one change per commit)
- Test after every change
- `git revert` immediately if anything breaks

### Testing Checklist
After each phase:
- [ ] PTT functionality works
- [ ] Audio transmission works
- [ ] Channel switching works
- [ ] UI updates correctly
- [ ] WebSocket broadcasts work

### Migration Order (Safest → Riskiest)
1. **Message display, status updates** (least critical)
2. **Channel switching, user lists** (medium risk)
3. **PTT and audio state** (most critical - do last)

## File-by-File Migration Plan

### Priority 1: Low Risk
- `client/webserver.go` - Message display functions
- Status update calls in `client/main.go`

### Priority 2: Medium Risk  
- `client/net.go` - Channel switching, connection status
- User list management

### Priority 3: High Risk (Do Last)
- `client/audio.go` - PTT state, audio levels
- Core audio pipeline integration

## Expected Outcome

### Before Refactor
```go
// Mixed concerns - BAD
func audioProcess() {
    // Audio work
    samples := readMic()
    
    // UI work mixed in
    WebTUISetLevel(getLevel(samples))
    WebTUIAddMessage("Processing audio")
    
    // Back to audio work  
    sendAudio(samples)
}
```

### After Refactor
```go
// Core - Pure audio logic
func audioProcess() {
    samples := readMic()
    appState.SetAudioLevel(getLevel(samples))
    sendAudio(samples)
}

// UI - Pure presentation logic
appState.OnAudioLevelChange(func(level int) {
    updateVisualization(level)  // Can experiment freely here
    addMessage("Processing audio")
})
```

### Benefits Achieved
- ✅ Core voice functionality = untouchable black box
- ✅ UI layer = safe experimentation zone  
- ✅ Independent testing of each layer
- ✅ Clear architectural boundaries
- ✅ Zero risk of breaking voice chat during UI work

## Success Criteria

1. **Functionality Preserved**: All current features work identically
2. **Clean Separation**: No direct calls from core systems to UI
3. **Observer Pattern**: UI responds to state changes only
4. **Independent Development**: Can modify UI without touching audio/network code
5. **Maintainable**: Clear boundaries between system responsibilities

**End Result**: A bulletproof voice chat core with a completely flexible UI layer.