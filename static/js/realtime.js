// Real-time WebSocket client for HTMX integration
class RealtimeClient {
    constructor(boardId, userId) {
        this.boardId = boardId;
        this.userId = userId;
        this.ws = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.connect();
    }

    connect() {
        const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${location.host}/ws/${this.boardId}`;
        
        this.ws = new WebSocket(wsUrl);
        
        this.ws.onopen = () => {
            console.log('Connected to real-time updates');
            this.reconnectAttempts = 0;
            this.sendPresenceUpdate('online');
        };
        
        this.ws.onmessage = (event) => {
            const message = JSON.parse(event.data);
            this.handleMessage(message);
        };
        
        this.ws.onclose = () => {
            console.log('WebSocket connection closed');
            this.handleReconnect();
        };
        
        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    }

    handleMessage(message) {
        switch (message.type) {
            case 'htmx_update':
                this.handleHTMXUpdate(message);
                break;
            case 'user_presence':
                this.handlePresenceUpdate(message);
                break;
            case 'cursor_move':
                this.handleCursorMove(message);
                break;
            case 'error':
                this.handleError(message);
                break;
        }
    }

    handleHTMXUpdate(message) {
        const target = document.querySelector(message.data.target);
        if (target) {
            // Update DOM using HTMX-style swapping
            switch (message.data.swap_strategy) {
                case 'outerHTML':
                    target.outerHTML = message.data.html_content;
                    break;
                case 'innerHTML':
                    target.innerHTML = message.data.html_content;
                    break;
                case 'beforeend':
                    target.insertAdjacentHTML('beforeend', message.data.html_content);
                    break;
            }
            
            // Trigger HTMX processing for new elements
            htmx.process(document.body);
        }
    }

    handlePresenceUpdate(message) {
        // Handle user presence updates (online/offline status)
        console.log('Presence update:', message);
    }

    handleCursorMove(message) {
        // Handle cursor movement from other users
        console.log('Cursor move:', message);
    }

    handleError(message) {
        console.error('WebSocket error message:', message);
    }

    sendPresenceUpdate(status) {
        this.send({
            type: 'user_presence',
            data: { status: status }
        });
    }

    sendTaskMove(taskId, columnId, position, version) {
        this.send({
            type: 'task_move',
            data: {
                task_id: taskId,
                column_id: columnId,
                position: position,
                version: version
            }
        });
    }

    sendCursorMove(x, y, element) {
        this.send({
            type: 'cursor_move',
            data: { x, y, element }
        });
    }

    send(message) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            message.board_id = this.boardId;
            message.user_id = this.userId;
            message.timestamp = new Date().toISOString();
            this.ws.send(JSON.stringify(message));
        }
    }

    handleReconnect() {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
            console.log(`Reconnecting in ${delay}ms...`);
            setTimeout(() => this.connect(), delay);
        }
    }
}

// Initialize real-time client when board page loads
document.addEventListener('DOMContentLoaded', function() {
    const boardElement = document.querySelector('[data-board-id]');
    const userElement = document.querySelector('[data-user-id]');
    
    if (boardElement && userElement) {
        const boardId = boardElement.dataset.boardId;
        const userId = userElement.dataset.userId;
        
        window.realtimeClient = new RealtimeClient(boardId, userId);
        
        // Integrate with existing drag-and-drop
        document.addEventListener('task-moved', function(event) {
            window.realtimeClient.sendTaskMove(
                event.detail.taskId,
                event.detail.columnId,
                event.detail.position,
                event.detail.version
            );
        });
        
        // Track cursor movement (throttled)
        let lastCursorUpdate = 0;
        document.addEventListener('mousemove', function(event) {
            const now = Date.now();
            if (now - lastCursorUpdate > 100) { // Throttle to 10fps
                lastCursorUpdate = now;
                window.realtimeClient.sendCursorMove(event.clientX, event.clientY, event.target.id);
            }
        });
    }
});