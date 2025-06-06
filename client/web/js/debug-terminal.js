// FILE: client/web/js/debug-terminal.js

const DebugTerminal = {
    // Simple initialization - no HTML creation
    init() {
        console.log('🔧 Debug Terminal initialized (export-only mode)');
    },

    // Just export the log directly
    exportLog() {
        const history = MessageRouter.getDebugHistory();
        let logContent = `AHCLI Debug Log - ${new Date().toISOString()}\n`;
        logContent += '='.repeat(50) + '\n\n';
        
        history.forEach(msg => {
            const timestamp = msg.timestamp || 'Unknown';
            const tag = msg.tag || '[INFO]';
            logContent += `[${timestamp}] ${tag} ${msg.content}\n`;
        });

        // Create download
        const blob = new Blob([logContent], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `ahcli-debug-${new Date().toISOString().slice(0, 19)}.log`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);

        console.log('🔧 Debug log exported successfully');
    },

    // Dummy functions for compatibility
    isOpen: () => false,
    addMessage: () => {},
    show: () => {},
    hide: () => {},
    toggle: () => {}
};