// FILE: client/web/js/message-router.js

// Message Router - Simple pass-through version
const MessageRouter = {
    // Initialize the router
    init() {
        console.log('🔀 Message Router initialized - simple pass-through mode');
    },
    
    // Main routing function - simplified
    route(messageData) {
        if (!messageData) return;
        
        const message = messageData.message || messageData.content || '';
        const msgType = messageData.type || 'info';
        
        console.log('🔀 Routing message:', { type: msgType, message: message });
        
        // Let everything pass through to the existing system
        // Chat messages will be processed by UserChat.update() from App.updateUI()
        
        // Only route non-chat messages to debug
        if (msgType !== 'chat') {
            this.handleDebugMessage(messageData);
        }
    },
    
    // Handle debug messages
    handleDebugMessage(messageData) {
        // Just update the legacy message area for important messages
        if (messageData.type === 'error') {
            const container = document.getElementById('messagesContainer');
            if (container) {
                const div = document.createElement('div');
                div.className = `message-item ${messageData.type}`;
                div.innerHTML = `
                    <div class="message-time">[${messageData.timestamp}]</div>
                    <div>${messageData.message}</div>
                `;
                container.appendChild(div);
                container.scrollTop = container.scrollHeight;
                
                // Keep only last 10 messages
                while (container.children.length > 10) {
                    container.removeChild(container.firstChild);
                }
            }
        }
    },
    
    // Send chat message to server
    sendChatMessage(message) {
        if (!App.state.connected) {
            console.log('💬 Cannot send - not connected');
            return;
        }
        
        // Send to server via existing command system
        App.sendCommand('chat', message);
        
        console.log('💬 Sent chat to server:', message);
    },
    
    // Dummy functions for compatibility
    onChannelChange() {},
    onConnectionChange() {},
    getDebugHistory() { return []; },
    getChatHistory() { return []; },
    getStats() { return {}; },
    clearHistory() {}
};