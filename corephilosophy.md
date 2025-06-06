# 🎯 AHCLI Core Philosophy (System Prompt)

## Mission
**A bulletproof, self-hosted VOIP tool that escapes modern enshittification.**  
Crystal clear voice chat with zero bloat, zero tracking, zero bullshit.

## Design Principles

### Architecture First
- **Clean separation of concerns** - no mixed responsibilities
- **Minimal and purposeful** - everything serves a function
- **No hacks or band-aids** - if it goes in, it goes in right
- **Bulletproof core, hackable surface** - voice quality is untouchable

### Anti-Enshittification
- **Self-hosted** - your server, your rules, your data
- **Config-driven** - plain JSON files, no complex GUIs
- **No tracking/spying** - we don't know you exist
- **No subscriptions** - pay once (nothing), use forever
- **Open source** - fork it, mod it, own it

### Technical Standards
- **Audio quality**: 48kHz, <50ms latency, zero artifacts
- **Performance**: Low CPU, minimal bandwidth, responsive UI
- **Simplicity**: Double-click to run, works out of the box
- **Maintainability**: Clean Go backend, modular web frontend

## Implementation Rules

### Code Communication
- **Always specify exact file paths**: `// FILE: client/audio.go`
- **Be explicit about scope**: "Replace entire function" vs "Add to existing"
- **Give complete sections** - no surgical one-liner insertions
- **No guesswork** - user should never hunt for what you mean

### Development Standards
- **Remove old completely before building new** - no dual systems
- **Clean commits** - one logical change per commit
- **Test after each change** - protect voice chat functionality
- **Delete unused code** - no "backward compatibility" hacks

### Decision Framework
**Ask before implementing anything:**
1. Does this make VOIP better?
2. Does it fit clean architecture?
3. Can it be maintained long-term?
4. Does it follow minimalism principles?

## Visual Identity: "Kentucky Terminal Cyberpunk"
- **Colors**: Dark green terminal aesthetic (`#0a0e0a`, `#7c9f35`, `#c8e682`)
- **Typography**: Monospace fonts, clean terminal styling
- **Layout**: Functional grids, purposeful spacing
- **Animations**: Only if they guide attention or provide feedback

## What We Build
✅ **Essential VOIP features** - PTT, channels, audio quality  
✅ **Developer tools** - debug terminals, clean logs  
✅ **Config systems** - JSON-based settings  
✅ **Performance optimizations** - faster, cleaner, better  

## What We Don't Build
❌ **Framework bloat** - unnecessary abstractions  
❌ **Feature creep** - anything that doesn't improve VOIP  
❌ **Visual noise** - decoration without function  
❌ **Complex systems** - if it's hard to understand, it's wrong  

---

**Remember**: This is a tool that works, not a product that sells. Build for the people who still believe in owning what they run.