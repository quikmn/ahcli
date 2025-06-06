// FILE: client/web/js/user-chat.js

// User Chat - Clean deduplication with self-message styling
const UserChat = {
    container: null,
    input: null,
    currentChannel: null,
    channelMessages: new Map(), // Store messages per channel
    processedMessageIds: new Set(), // Track processed message IDs globally
    lastAppStateMessageCount: 0, // Track appState messages processed
    
    // Initialize user chat
    init() {
        this.container = document.getElementById('userChatContainer') ||
            document.querySelector('#messages-container .user-chat-container');
        this.input = document.getElementById('chatInput') ||
            document.querySelector('#messages-container .chat-input');
            
        if (this.input) {
            this.setupChatInput();
        }
        
        // Set initial channel
        this.currentChannel = 'General';
        
        // Welcome message
        this.addSystemMessage('💬 AHCLI Chat System Ready');
        
        console.log('💬 User Chat initialized with self-message styling');
    },
    
    // Set up chat input handling
    setupChatInput() {
        this.input.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                const message = e.target.value.trim();
                if (message) {
                    this.sendMessage(message);
                    e.target.value = '';
                }
            }
        });
        
        this.updateInputPlaceholder();
    },
    
    // Update input placeholder
    updateInputPlaceholder() {
        if (this.input && this.currentChannel) {
            this.input.placeholder = `Type message in #${this.currentChannel}...`;
        } else if (this.input) {
            this.input.placeholder = 'Connect to server to chat...';
        }
    },
    
    // Send message to server
    sendMessage(message) {
        if (!App.state.connected) {
            this.addSystemMessage('⚠️ Cannot send chat: not connected to server');
            return;
        }
        
        // Send via existing command system
        App.sendCommand('chat', message);
        
        console.log('💬 Sent chat message:', message);
    },
    
    // Create unique message ID for deduplication
    createMessageId(messageText, channel) {
        // Create ID from content + channel, ignoring timestamp differences
        const cleanText = messageText.replace(/\[\d{2}:\d{2}(?::\d{2})?\]/g, ''); // Remove timestamps
        const match = cleanText.match(/<([^>]+)>\s*(.+)/); // Extract <username> message
        
        if (match) {
            const [, username, content] = match;
            return `${channel}-${username}-${content.trim()}`;
        }
        
        return `${channel}-${cleanText.trim()}`;
    },
    
    // Extract username from formatted message
    extractUsername(messageText) {
        const match = messageText.match(/<([^>]+)>/);
        return match ? match[1] : null;
    },
    
    // Check if message is from current user
    isOwnMessage(messageText) {
        const username = this.extractUsername(messageText);
        const currentUser = App.state.nickname;
        return username === currentUser;
    },
    
    // Process messages from App state - ONLY NEW ONES
    update(newState) {
        // Handle channel changes
        if (newState.currentChannel && newState.currentChannel !== this.currentChannel) {
            this.switchToChannel(newState.currentChannel);
        }
        
        // Process ONLY truly new messages from appState
        if (newState.messages && newState.messages.length > this.lastAppStateMessageCount) {
            const newMessages = newState.messages.slice(this.lastAppStateMessageCount);
            
            console.log(`💬 Processing ${newMessages.length} NEW messages from appState`);
            
            newMessages.forEach(msg => {
                if (msg.type === 'chat') {
                    this.processNewChatMessage(msg.message);
                }
            });
            
            this.lastAppStateMessageCount = newState.messages.length;
        }
        
        // Update input state
        if (!newState.connected && this.input) {
            this.input.placeholder = 'Connect to server to chat...';
            this.input.disabled = true;
        } else if (this.input) {
            this.input.disabled = false;
            this.updateInputPlaceholder();
        }
    },
    
    // Process a new chat message with deduplication
    processNewChatMessage(messageText) {
        const messageId = this.createMessageId(messageText, this.currentChannel);
        
        // Check if we've already processed this message
        if (this.processedMessageIds.has(messageId)) {
            console.log('💬 Skipping duplicate message ID:', messageId);
            return;
        }
        
        // Mark as processed
        this.processedMessageIds.add(messageId);
        
        // Clean up old IDs (keep last 1000)
        if (this.processedMessageIds.size > 1000) {
            const oldIds = Array.from(this.processedMessageIds).slice(0, 500);
            oldIds.forEach(id => this.processedMessageIds.delete(id));
        }
        
        // Add to current channel storage
        this.addChatMessageToChannel(messageText, this.currentChannel);
        
        console.log('💬 Processed new chat message:', messageText);
    },
    
    // Add chat message to specific channel
    addChatMessageToChannel(messageText, channel) {
        const targetChannel = channel || this.currentChannel || 'General';
        
        // Store in channel-specific storage
        if (!this.channelMessages.has(targetChannel)) {
            this.channelMessages.set(targetChannel, []);
        }
        
        const channelMsgs = this.channelMessages.get(targetChannel);
        channelMsgs.push(messageText);
        
        // Keep only last 100 messages per channel
        if (channelMsgs.length > 100) {
            channelMsgs.splice(0, channelMsgs.length - 100);
        }
        
        // Display if it's for current channel
        if (targetChannel === this.currentChannel) {
            this.displayMessage(messageText);
        }
        
        console.log(`💬 Added message to ${targetChannel} (${channelMsgs.length} total)`);
    },
    
    // Display a message in the UI with proper styling
    displayMessage(messageText) {
        if (!this.container) return;
        
        const chatLine = document.createElement('div');
        
        // Check if this is the current user's message
        const isOwn = this.isOwnMessage(messageText);
        
        // Apply appropriate CSS classes
        if (isOwn) {
            chatLine.className = 'chat-line chat-line-self';
        } else {
            chatLine.className = 'chat-line chat-line-other';
        }
        
        // Parse the message for better formatting
        const timestampMatch = messageText.match(/\[(\d{2}:\d{2})\]/);
        const userMatch = messageText.match(/<([^>]+)>/);
        const contentMatch = messageText.match(/>\s*(.+)$/);
        
        if (timestampMatch && userMatch && contentMatch) {
            const [, timestamp] = timestampMatch;
            const [, username] = userMatch;
            const [, content] = contentMatch;
            
            if (isOwn) {
                // Your messages: emphasized styling
                chatLine.innerHTML = `
                    <span class="chat-timestamp-self">[${timestamp}]</span>
                    <span class="chat-separator"> </span>
                    <span class="chat-username-self">&lt;${username}&gt;</span>
                    <span class="chat-separator"> </span>
                    <span class="chat-message-self">${content}</span>
                `;
            } else {
                // Other users: standard styling
                chatLine.innerHTML = `
                    <span class="chat-timestamp">[${timestamp}]</span>
                    <span class="chat-separator"> </span>
                    <span class="chat-username">&lt;${username}&gt;</span>
                    <span class="chat-separator"> </span>
                    <span class="chat-message">${content}</span>
                `;
            }
        } else {
            // Fallback for malformed messages
            chatLine.innerHTML = `<span class="chat-message">${messageText}</span>`;
        }
        
        this.container.appendChild(chatLine);
        this.scrollToBottom();
    },
    
    // Switch to different channel - CLEAN IMPLEMENTATION
    switchToChannel(newChannel) {
        if (newChannel === this.currentChannel) return;
        
        console.log(`💬 Switching chat: ${this.currentChannel} → ${newChannel}`);
        
        this.currentChannel = newChannel;
        
        // Clear display
        if (this.container) {
            this.container.innerHTML = '';
        }
        
        // Add channel notification
        this.addChannelNotification(newChannel);
        
        // Load messages for new channel (NO re-processing)
        this.loadChannelMessages(newChannel);
        
        // Update placeholder
        this.updateInputPlaceholder();
    },
    
    // Load messages for a channel - DISPLAY ONLY
    loadChannelMessages(channel) {
        const messages = this.channelMessages.get(channel) || [];
        
        console.log(`💬 Loading ${messages.length} stored messages for ${channel}`);
        
        // Just display stored messages - no processing
        messages.forEach(msg => {
            this.displayMessage(msg);
        });
        
        if (messages.length === 0) {
            this.addSystemMessage(`Welcome to #${channel}`);
        }
    },
    
    // Add system message
    addSystemMessage(message) {
        if (!this.container) return;
        
        const chatLine = document.createElement('div');
        chatLine.className = 'chat-line chat-line-system';
        
        const timestamp = new Date().toLocaleTimeString('en-US', {
            hour12: false, 
            hour: '2-digit', 
            minute: '2-digit'
        });
        
        chatLine.innerHTML = `
            <span class="chat-timestamp-system">[${timestamp}]</span>
            <span class="chat-separator"> </span>
            <span class="chat-username-system">&lt;System&gt;</span>
            <span class="chat-separator"> </span>
            <span class="chat-message-system">${message}</span>
        `;
        
        this.container.appendChild(chatLine);
        this.scrollToBottom();
    },
    
    // Add channel notification
    addChannelNotification(channel) {
        if (!this.container) return;
        
        const chatLine = document.createElement('div');
        chatLine.className = 'chat-line chat-line-notification';
        
        const timestamp = new Date().toLocaleTimeString('en-US', {
            hour12: false, 
            hour: '2-digit', 
            minute: '2-digit'
        });
        
        chatLine.innerHTML = `
            <span class="chat-timestamp-notification">[${timestamp}]</span>
            <span class="chat-separator"> </span>
            <span class="chat-notification">*** You joined #${channel} ***</span>
        `;
        
        this.container.appendChild(chatLine);
        this.scrollToBottom();
    },
    
    // Scroll to bottom
    scrollToBottom() {
        if (this.container) {
            this.container.scrollTop = this.container.scrollHeight;
        }
    },
    
    // Focus input
    focus() {
        if (this.input) {
            this.input.focus();
        }
    },
    
    // Get stats
    getStats() {
        return {
            currentChannel: this.currentChannel,
            channelCount: this.channelMessages.size,
            totalMessages: Array.from(this.channelMessages.values()).reduce((sum, msgs) => sum + msgs.length, 0),
            processedIds: this.processedMessageIds.size,
            lastAppStateCount: this.lastAppStateMessageCount
        };
    }
};