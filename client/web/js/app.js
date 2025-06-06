// FILE: client/web/js/app.js

// Main Application Controller with Chat Integration
const App = {
    state: {},
    processedMessageCount: 0,
    previousChannel: null,
    
    // Initialize the application
    async init() {
        console.log('🚀 AHCLI UI Starting with Chat System...');
        try {
            // Load all components
            await this.loadComponents();
            
            // Initialize WebSocket connection
            WebSocketManager.init();
            
            // Initialize audio visualization
            AudioViz.init();
            
            // Set up event handlers
            this.setupEventHandlers();
            
            // Initialize chat systems (order matters!)
            MessageRouter.init();
            UserChat.init();
            DebugTerminal.init();
            
            console.log('✅ AHCLI UI Ready with Multi-User Chat');
        } catch (error) {
            console.error('❌ Failed to initialize AHCLI:', error);
        }
    },
    
    // Load HTML components into containers
    async loadComponents() {
        const components = [
            { container: 'sidebar-container', file: 'components/sidebar.html' },
            { container: 'messages-container', file: 'components/messages.html' },
            { container: 'channels-container', file: 'components/channels.html' },
            { container: 'footer-container', file: 'components/footer.html' }
        ];
        
        for (const component of components) {
            try {
                const response = await fetch(component.file);
                if (!response.ok) throw new Error(`Failed to load ${component.file}`);
                
                const html = await response.text();
                const container = document.getElementById(component.container);
                if (container) {
                    container.innerHTML = html;
                    console.log(`✅ Loaded ${component.file}`);
                } else {
                    console.error(`❌ Container '${component.container}' not found`);
                }
            } catch (error) {
                console.error(`❌ Failed to load ${component.file}:`, error);
            }
        }
    },
    
    // Set up global event handlers
    setupEventHandlers() {
        // Wait for components to be loaded before setting up handlers
        setTimeout(() => {
            this.setupCommandInput();
            this.setupUptimeTimer();
            this.setupDebugButton();
            this.setupChatKeyboardShortcuts();
        }, 100);
    },
    
    // Set up command input handling
    setupCommandInput() {
        const commandInput = document.getElementById('commandInput');
        if (commandInput) {
            commandInput.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') {
                    const input = e.target.value.trim();
                    if (input) {
                        this.handleCommand(input);
                        e.target.value = '';
                    }
                }
            });
            console.log('✅ Command input handler set up');
        } else {
            console.warn('⚠️ Command input not found, retrying...');
            // Retry after components are fully loaded
            setTimeout(() => this.setupCommandInput(), 500);
        }
    },
    
    // Set up keyboard shortcuts for chat
    setupChatKeyboardShortcuts() {
        document.addEventListener('keydown', (e) => {
            // Enter key focuses chat input (if not already focused on an input)
            if (e.key === 'Enter' && !['INPUT', 'TEXTAREA'].includes(e.target.tagName)) {
                e.preventDefault();
                UserChat.focus();
            }
            
            // Escape key clears chat input focus
            if (e.key === 'Escape') {
                document.activeElement.blur();
            }
        });
        
        console.log('✅ Chat keyboard shortcuts set up');
    },
    
    // Set up uptime timer
    setupUptimeTimer() {
        // Uptime counter
        setInterval(() => {
            this.updateUptime();
        }, 1000);
    },
    
    setupDebugButton() {
        const debugBtn = document.querySelector('.debug-terminal-btn');
        if (debugBtn) {
            debugBtn.onclick = () => {
                DebugTerminal.exportLog();
            };
            debugBtn.textContent = '📁 Export Log'; // Updated button text
            console.log('✅ Debug button set up for log export');
        }
    },
    
    // Handle terminal commands
    handleCommand(input) {
        if (!input.startsWith('/')) {
            // Not a command - treat as chat message
            if (this.state.connected) {
                MessageRouter.sendChatMessage(input);
            } else {
                console.log('Not connected to server for chat');
            }
            return;
        }
        
        const parts = input.slice(1).split(' ');
        const command = parts[0];
        const args = parts.slice(1).join(' ');
        
        console.log(`Executing command: /${command} ${args}`);
        this.sendCommand(command, args);
    },
    
    // Send command to server
    sendCommand(command, args = '') {
        fetch('/api/command', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ command, args })
        }).catch(error => {
            console.error('Failed to send command:', error);
        });
    },
    
    // Update UI based on state changes
    updateUI(newState) {
        console.log('🔄 updateUI called with messages:', newState.messages?.length || 0);
        
        // Check for channel changes
        if (this.state.currentChannel !== newState.currentChannel) {
            MessageRouter.onChannelChange(this.state.currentChannel, newState.currentChannel);
            this.previousChannel = this.state.currentChannel;
        }
        
        // Check for connection changes
        if (this.state.connected !== newState.connected) {
            MessageRouter.onConnectionChange(newState.connected);
        }
        
        this.state = newState;
        
        // Update connection status
        this.updateConnectionStatus();
        
        // Update user info
        this.updateUserInfo();
        
        // Update network stats
        this.updateNetworkStats();
        
        // Update PTT status
        this.updatePTTStatus();
        
        // Process messages through MessageRouter (with deduplication)
        if (newState.messages) {
            const processedCount = this.processedMessageCount || 0;
            const newMessages = newState.messages.slice(processedCount);
            
            if (newMessages.length > 0) {
                console.log('🔄 Processing NEW messages:', newMessages.length);
                newMessages.forEach(msg => {
                    console.log('🔄 Routing NEW message:', msg);
                    MessageRouter.route({
                        message: msg.message,
                        timestamp: msg.timestamp,
                        type: msg.type
                    });
                });
                this.processedMessageCount = newState.messages.length;
            }
        }
        
        // Update channels
        this.updateChannels();
        
        // Update audio visualization
        AudioViz.update(newState);
        
        // Update user chat
        UserChat.update(newState);
    },
    
    // Update connection status indicator
    updateConnectionStatus() {
        const statusDot = document.getElementById('connectionStatus');
        const statusText = document.getElementById('statusText');
        
        if (this.state.connected) {
            statusDot?.classList.add('connected');
            if (statusText) statusText.textContent = `Connected to ${this.state.serverName}`;
        } else {
            statusDot?.classList.remove('connected');
            if (statusText) statusText.textContent = 'Disconnected';
        }
    },
    
    // Update user information
    updateUserInfo() {
        const nickname = document.getElementById('nickname');
        const currentChannel = document.getElementById('currentChannel');
        
        if (nickname) nickname.textContent = this.state.nickname || '-';
        if (currentChannel) currentChannel.textContent = this.state.currentChannel || 'None';
    },
    
    // Update network statistics
    updateNetworkStats() {
        const packetsRx = document.getElementById('packetsRx');
        const packetsTx = document.getElementById('packetsTx');
        const pttKeyText = document.getElementById('pttKeyText');
        
        if (packetsRx) packetsRx.textContent = this.state.packetsRx || 0;
        if (packetsTx) packetsTx.textContent = this.state.packetsTx || 0;
        if (pttKeyText) pttKeyText.textContent = `Hold ${this.state.pttKey || 'LSHIFT'} to transmit`;
    },
    
    // Update PTT status and audio bar
    updatePTTStatus() {
        const pttIndicator = document.getElementById('pttIndicator');
        const pttText = document.getElementById('pttText');
        
        if (this.state.pttActive) {
            pttIndicator?.classList.add('active');
            if (pttText) pttText.textContent = 'Transmitting';
        } else {
            pttIndicator?.classList.remove('active');
            if (pttText) pttText.textContent = 'Ready';
        }
        
        // Update audio bar
        this.updateAudioBar(this.state.audioLevel || 0);
    },
    
    // Update channels and users
    updateChannels() {
        const container = document.getElementById('channelsContainer');
        if (!container || !this.state.channels) return;
        
        container.innerHTML = '';
        
        this.state.channels.forEach(channel => {
            // Channel header
            const channelDiv = document.createElement('div');
            channelDiv.className = `channel-item ${channel === this.state.currentChannel ? 'active' : ''}`;
            channelDiv.innerHTML = `
                <span class="channel-icon">${channel === this.state.currentChannel ? '▶' : '▷'}</span>
                ${channel}
            `;
            channelDiv.onclick = () => this.joinChannel(channel);
            container.appendChild(channelDiv);
            
            // Channel users
            if (this.state.channelUsers && this.state.channelUsers[channel]) {
                this.state.channelUsers[channel].forEach(user => {
                    const userDiv = document.createElement('div');
                    userDiv.className = `user-item ${user === this.state.nickname ? 'self' : ''}`;
                    userDiv.innerHTML = `├─ ${user}${user === this.state.nickname ? ' (you)' : ''}`;
                    container.appendChild(userDiv);
                });
            } else if (channel === this.state.currentChannel && this.state.nickname) {
                // Show yourself in current channel if no user list
                const userDiv = document.createElement('div');
                userDiv.className = 'user-item self';
                userDiv.innerHTML = `├─ ${this.state.nickname} (you)`;
                container.appendChild(userDiv);
            }
        });
    },
    
    // Update audio level bar
    updateAudioBar(level) {
        const audioBar = document.getElementById('audioBar');
        if (!audioBar) return;
        
        // Create segments if they don't exist
        if (audioBar.children.length === 0) {
            for (let i = 0; i < 20; i++) {
                const segment = document.createElement('div');
                segment.className = 'audio-bar-segment';
                segment.style.height = `${4 + i}px`;
                audioBar.appendChild(segment);
            }
        }
        
        // Update segment states
        const segments = audioBar.children;
        const activeSegments = Math.floor((level / 100) * segments.length);
        
        for (let i = 0; i < segments.length; i++) {
            segments[i].classList.remove('active', 'high', 'peak');
            if (i < activeSegments) {
                if (i < segments.length * 0.6) {
                    segments[i].classList.add('active');
                } else if (i < segments.length * 0.8) {
                    segments[i].classList.add('high');
                } else {
                    segments[i].classList.add('peak');
                }
            }
        }
    },
    
    // Update uptime display
    updateUptime() {
        if (!this.state.connected || !this.state.connectionTime) return;
        
        const uptimeElement = document.getElementById('uptime');
        if (!uptimeElement) return;
        
        const uptimeSeconds = Math.floor((Date.now() - new Date(this.state.connectionTime).getTime()) / 1000);
        uptimeElement.textContent = this.formatUptime(uptimeSeconds);
    },
    
    // Format uptime string
    formatUptime(seconds) {
        const hours = Math.floor(seconds / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        const secs = seconds % 60;
        
        if (hours > 0) {
            return `${hours}h ${minutes}m ${secs}s`;
        } else if (minutes > 0) {
            return `${minutes}m ${secs}s`;
        } else {
            return `${secs}s`;
        }
    },
    
    // Join a channel
    joinChannel(channel) {
        if (channel !== this.state.currentChannel) {
            this.sendCommand('join', channel);
            console.log(`💬 Switching to channel: ${channel}`);
        }
    },
    
    // Get router statistics for debugging
    getRouterStats() {
        return MessageRouter.getStats();
    }
};