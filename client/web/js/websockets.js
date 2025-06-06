// WebSocket Connection Manager
const WebSocketManager = {
    ws: null,
    reconnectAttempts: 0,
    maxReconnectAttempts: 10,
    reconnectDelay: 1000,
    
    // Initialize WebSocket connection
    init() {
        this.connect();
    },
    
    // Connect to WebSocket
    connect() {
        const wsUrl = `ws://${window.location.host}/ws`;
        
        try {
            this.ws = new WebSocket(wsUrl);
            this.setupEventHandlers();
            console.log('🔌 Attempting WebSocket connection...');
        } catch (error) {
            console.error('❌ WebSocket connection failed:', error);
            this.scheduleReconnect();
        }
    },
    
    // Set up WebSocket event handlers
    setupEventHandlers() {
        this.ws.onopen = () => {
            console.log('✅ WebSocket connected');
            this.reconnectAttempts = 0;
        };
        
        this.ws.onmessage = (event) => {
            try {
                const state = JSON.parse(event.data);
                App.updateUI(state);
            } catch (error) {
                console.error('❌ Failed to parse WebSocket message:', error);
            }
        };
        
        this.ws.onclose = (event) => {
            console.log('🔌 WebSocket disconnected:', event.code, event.reason);
            this.scheduleReconnect();
        };
        
        this.ws.onerror = (error) => {
            console.error('❌ WebSocket error:', error);
        };
    },
    
    // Schedule reconnection attempt
    scheduleReconnect() {
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            console.error('❌ Max reconnection attempts reached');
            return;
        }
        
        this.reconnectAttempts++;
        const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1); // Exponential backoff
        
        console.log(`🔄 Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
        
        setTimeout(() => {
            this.connect();
        }, delay);
    },
    
    // Send message through WebSocket
    send(data) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(data));
        } else {
            console.warn('⚠️ WebSocket not connected, cannot send message');
        }
    },
    
    // Close WebSocket connection
    close() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
    }
};