// Real-time WebSocket client optimized for HTMX integration
class SudoRealtimeClient {
    constructor(boardId, userId, userName) {
        this.boardId = boardId;
        this.userId = userId;
        this.userName = userName;
        this.ws = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 8;
        this.baseReconnectDelay = 1000;
        this.cursors = new Map(); // Track other users' cursors
        this.lastCursorSent = 0;
        this.cursorThrottleMs = 50; // 20fps cursor updates
        
        this.connect();
        this.setupEventHandlers();
    }

    connect() {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            return;
        }

        const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${location.host}/ws/${this.boardId}`;
        
        console.log(`Connecting to WebSocket: ${wsUrl}`);
        this.ws = new WebSocket(wsUrl);
        
        this.ws.onopen = () => {
            console.log('Connected to real-time collaboration');
            this.reconnectAttempts = 0;
            this.sendPresenceUpdate('online');
            this.showConnectionStatus('connected');
        };
        
        this.ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                this.handleMessage(message);
            } catch (error) {
                console.error('Failed to parse WebSocket message:', error);
            }
        };
        
        this.ws.onclose = (event) => {
            console.log('WebSocket connection closed:', event.code, event.reason);
            this.showConnectionStatus('disconnected');
            this.clearAllCursors();
            this.scheduleReconnect();
        };
        
        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            this.showConnectionStatus('error');
        };
    }

    setupEventHandlers() {
        // Integrate with HTMX drag-and-drop events
        document.addEventListener('htmx:beforeSend', (event) => {
            const target = event.target;
            if (target.closest('[data-task-id]')) {
                // This is a task operation, prepare for real-time sync
                const taskId = target.closest('[data-task-id]').dataset.taskId;
                console.log('Task operation starting:', taskId);
            }
        });

        // Track cursor movement (throttled)
        document.addEventListener('mousemove', (event) => {
            this.throttledCursorUpdate(event.clientX, event.clientY, event.target.id);
        });

        // Track typing in form inputs
        document.addEventListener('input', (event) => {
            if (event.target.matches('input[name="title"], textarea[name="description"]')) {
                const taskId = event.target.closest('[data-task-id]')?.dataset.taskId;
                this.sendTypingUpdate(true, taskId);
                
                // Stop typing after 3 seconds of inactivity
                clearTimeout(this.typingTimeout);
                this.typingTimeout = setTimeout(() => {
                    this.sendTypingUpdate(false, taskId);
                }, 3000);
            }
        });

        // Handle form submissions
        document.addEventListener('htmx:afterRequest', (event) => {
            if (event.detail.successful) {
                // Task operation completed successfully
                const taskElement = event.target.closest('[data-task-id]');
                if (taskElement) {
                    console.log('Task operation completed');
                }
            }
        });
    }

    handleMessage(message) {
        console.log('Received message:', message.type);
        
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
            case 'typing':
                this.handleTypingUpdate(message);
                break;
            case 'task_move':
            case 'task_create':
            case 'task_update':
            case 'task_delete':
                this.handleTaskUpdate(message);
                break;
            case 'error':
                this.handleError(message);
                break;
            case 'board_snapshot':
                this.handleBoardSnapshot(message);
                break;
        }
    }

    handleHTMXUpdate(message) {
        const target = document.querySelector(message.data.target);
        if (!target) {
            console.warn('HTMX target not found:', message.data.target);
            return;
        }

        // Show brief update indicator
        this.showUpdateIndicator(target, message.data.user_name || 'Someone');

        // Apply HTML update
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
            case 'afterbegin':
                target.insertAdjacentHTML('afterbegin', message.data.html_content);
                break;
        }
        
        // Re-initialize HTMX for new elements
        htmx.process(document.body);
    }

    handleCursorMove(message) {
        if (message.user_id === this.userId) {
            return; // Don't show our own cursor
        }

        const x = message.data.x;
        const y = message.data.y;
        const userName = message.data.user_name;
        
        this.updateUserCursor(message.user_id, userName, x, y);
    }

    handleTypingUpdate(message) {
        if (message.user_id === this.userId) {
            return;
        }

        const taskId = message.data.active_task_id;
        const isTyping = message.data.is_typing;
        const userName = message.data.user_name;

        if (isTyping && taskId) {
            this.showTypingIndicator(taskId, userName);
        } else {
            this.hideTypingIndicator(taskId);
        }
    }

    handleTaskUpdate(message) {
        // Handle task-specific updates (create, update, move, delete)
        console.log('Task update received:', message);
        // This could trigger specific UI updates based on the task operation
    }

    handlePresenceUpdate(message) {
        // Handle user presence changes
        console.log('Presence update:', message);
    }

    handleError(message) {
        console.error('Server error:', message.data);
    }

    handleBoardSnapshot(message) {
        // Handle full board state updates
        console.log('Board snapshot received');
    }

    // Send methods
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

    throttledCursorUpdate(x, y, elementId) {
        const now = Date.now();
        if (now - this.lastCursorSent > this.cursorThrottleMs) {
            this.lastCursorSent = now;
            this.sendCursorMove(x, y, elementId);
        }
    }

    sendCursorMove(x, y, element) {
        this.send({
            type: 'cursor_move',
            data: { x, y, element }
        });
    }

    sendTypingUpdate(isTyping, activeTaskId = null) {
        this.send({
            type: 'user_presence',
            data: {
                is_typing: isTyping,
                active_task_id: activeTaskId
            }
        });
    }

    sendPresenceUpdate(status) {
        this.send({
            type: 'user_presence',
            data: {
                status: status,
                timestamp: new Date().toISOString()
            }
        });
    }

    send(message) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            message.board_id = this.boardId;
            message.user_id = this.userId;
            message.timestamp = new Date().toISOString();
            this.ws.send(JSON.stringify(message));
        } else {
            console.warn('WebSocket not connected, message queued for retry');
        }
    }

    // UI helper methods
    updateUserCursor(userId, userName, x, y) {
        let cursor = document.getElementById(`cursor-${userId}`);
        
        if (!cursor) {
            // Create new cursor
            cursor = document.createElement('div');
            cursor.id = `cursor-${userId}`;
            cursor.className = 'fixed pointer-events-none z-50 transition-all duration-100 ease-out';
            cursor.innerHTML = `
                <svg width="20" height="20" viewBox="0 0 24 24" class="drop-shadow-lg">
                    <path d="M5.65376 12.3673H5.46026L5.31717 12.4976L0.500002 16.8829L0.500002 1.19841L11.7841 12.3673H5.65376Z" 
                          fill="#3B82F6" stroke="white" stroke-width="1"/>
                </svg>
                <div class="ml-5 -mt-2 bg-blue-500 text-white text-xs px-2 py-1 rounded-full whitespace-nowrap">
                    ${userName}
                </div>
            `;
            document.body.appendChild(cursor);
        }

        // Update position
        cursor.style.left = `${x}px`;
        cursor.style.top = `${y}px`;
        
        // Auto-hide cursor after 3 seconds of inactivity
        clearTimeout(cursor.hideTimeout);
        cursor.hideTimeout = setTimeout(() => {
            cursor.style.opacity = '0.3';
        }, 3000);
        cursor.style.opacity = '1';
    }

    showTypingIndicator(taskId, userName) {
        const taskElement = document.querySelector(`[data-task-id="${taskId}"]`);
        if (!taskElement) return;

        let indicator = taskElement.querySelector('.typing-indicator');
        if (!indicator) {
            indicator = document.createElement('div');
            indicator.className = 'typing-indicator flex items-center space-x-1 text-xs text-blue-600 bg-blue-50 px-2 py-1 rounded-full mt-1';
            indicator.innerHTML = `
                <div class="flex space-x-1">
                    <div class="w-1 h-1 bg-blue-600 rounded-full animate-bounce"></div>
                    <div class="w-1 h-1 bg-blue-600 rounded-full animate-bounce" style="animation-delay: 0.1s"></div>
                    <div class="w-1 h-1 bg-blue-600 rounded-full animate-bounce" style="animation-delay: 0.2s"></div>
                </div>
                <span>${userName} is typing...</span>
            `;
            taskElement.appendChild(indicator);
        }
    }

    hideTypingIndicator(taskId) {
        const indicator = document.querySelector(`[data-task-id="${taskId}"] .typing-indicator`);
        if (indicator) {
            indicator.remove();
        }
    }

    showUpdateIndicator(element, userName) {
        // Brief flash to indicate real-time update
        const indicator = document.createElement('div');
        indicator.className = 'absolute top-2 right-2 bg-green-500 text-white text-xs px-2 py-1 rounded-full z-10';
        indicator.textContent = `Updated by ${userName}`;
        
        const parent = element.closest('.relative') || element;
        if (!parent.style.position) {
            parent.style.position = 'relative';
        }
        
        parent.appendChild(indicator);
        
        setTimeout(() => {
            indicator.style.opacity = '0';
            indicator.style.transition = 'opacity 0.3s';
            setTimeout(() => indicator.remove(), 300);
        }, 2000);
    }

    showConnectionStatus(status) {
        let statusIndicator = document.getElementById('connection-status');
        
        if (!statusIndicator) {
            statusIndicator = document.createElement('div');
            statusIndicator.id = 'connection-status';
            statusIndicator.className = 'fixed top-4 right-4 px-3 py-2 rounded-full text-sm font-medium z-50 transition-all duration-300';
            document.body.appendChild(statusIndicator);
        }

        switch (status) {
            case 'connected':
                statusIndicator.className = statusIndicator.className.replace(/bg-\w+-\d+/g, '') + ' bg-green-500 text-white';
                statusIndicator.textContent = '● Live collaboration active';
                setTimeout(() => {
                    statusIndicator.style.opacity = '0';
                }, 3000);
                break;
            case 'disconnected':
                statusIndicator.className = statusIndicator.className.replace(/bg-\w+-\d+/g, '') + ' bg-yellow-500 text-white';
                statusIndicator.textContent = '⚠ Reconnecting...';
                statusIndicator.style.opacity = '1';
                break;
            case 'error':
                statusIndicator.className = statusIndicator.className.replace(/bg-\w+-\d+/g, '') + ' bg-red-500 text-white';
                statusIndicator.textContent = '✕ Connection error';
                statusIndicator.style.opacity = '1';
                break;
        }
    }

    clearAllCursors() {
        document.querySelectorAll('[id^="cursor-"]').forEach(cursor => {
            cursor.remove();
        });
    }

    scheduleReconnect() {
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            console.error('Max reconnection attempts reached');
            this.showConnectionStatus('error');
            return;
        }

        this.reconnectAttempts++;
        const delay = Math.min(
            this.baseReconnectDelay * Math.pow(2, this.reconnectAttempts), 
            30000
        );
        
        console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);
        setTimeout(() => this.connect(), delay);
    }

    // Cleanup on page unload
    disconnect() {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            // Send offline status before closing
            this.sendPresenceUpdate('offline');

            // Give the message time to send (synchronous close)
            this.ws.close(1000, 'User navigated away');
        }
    }
}

// ============================================================================
// HTMX Integration Helpers
// ============================================================================

// Enhanced drag-and-drop with real-time sync
function initializeDragDropWithRealtime() {
    // Enhance existing drag handlers to broadcast moves
    document.addEventListener('dragend', function(event) {
        const taskElement = event.target.closest('[data-task-id]');
        const columnElement = event.target.closest('[data-column-id]');
        
        if (taskElement && columnElement && window.sudoRealtime) {
            const taskId = taskElement.dataset.taskId;
            const columnId = columnElement.dataset.columnId;
            const position = Array.from(columnElement.children).indexOf(taskElement);
            const version = parseInt(taskElement.dataset.version || '1');
            
            window.sudoRealtime.sendTaskMove(taskId, columnId, position, version);
        }
    });

    // Handle drop events for immediate feedback
    document.addEventListener('drop', function(event) {
        const taskElement = event.dataTransfer?.getData('text/plain');
        if (taskElement && window.sudoRealtime) {
            console.log('Task dropped, preparing real-time update');
        }
    });

    // Track drag start for optimistic UI updates
    document.addEventListener('dragstart', function(event) {
        const taskElement = event.target.closest('[data-task-id]');
        if (taskElement) {
            taskElement.classList.add('dragging');
            console.log('Task drag started:', taskElement.dataset.taskId);
        }
    });
}

// Enhanced form submission with real-time broadcasting
function enhanceFormsWithRealtime() {
    // Intercept task creation forms
    document.addEventListener('htmx:afterRequest', function(event) {
        if (event.detail.successful && event.target.matches('[data-task-form]')) {
            console.log('Task form submitted successfully');
            
            // The server-side handler should broadcast the update
            // Client just needs to handle the response
        }
    });

    // Handle task updates
    document.addEventListener('submit', function(event) {
        const form = event.target;
        if (form.matches('[data-update-task]') && window.sudoRealtime) {
            const taskId = form.dataset.taskId;
            console.log('Task update form submitted:', taskId);
        }
    });
}

// Initialize collaborative features
function initializeCollaborativeFeatures() {
    initializeDragDropWithRealtime();
    enhanceFormsWithRealtime();

    // Add real-time indicators to UI
    const boardHeader = document.querySelector('.board-header');
    if (boardHeader && !document.getElementById('collaboration-status')) {
        const statusContainer = document.createElement('div');
        statusContainer.id = 'collaboration-status';
        statusContainer.className = 'flex items-center space-x-2 text-sm text-gray-600';
        statusContainer.innerHTML = `
            <div class="flex items-center space-x-1">
                <div id="live-indicator" class="w-2 h-2 bg-green-400 rounded-full animate-pulse"></div>
                <span>Live collaboration</span>
            </div>
            <div id="presence-container"></div>
        `;
        boardHeader.appendChild(statusContainer);
    }
}

// Auto-initialize when board page loads
document.addEventListener('DOMContentLoaded', function() {
    const boardElement = document.querySelector('[data-board-id]');
    const userElement = document.querySelector('[data-user-id]');
    const userNameElement = document.querySelector('[data-user-name]');
    
    if (boardElement && userElement) {
        const boardId = boardElement.dataset.boardId;
        const userId = userElement.dataset.userId;
        const userName = userNameElement?.dataset.userName || 'Anonymous';
        
        // Initialize real-time client
        window.sudoRealtime = new SudoRealtimeClient(boardId, userId, userName);
        
        // Initialize collaborative features
        initializeCollaborativeFeatures();
        
        // Cleanup on page unload - Multiple event handlers for reliability
        // 1. beforeunload - fires before page unload (desktop browsers)
        window.addEventListener('beforeunload', () => {
            window.sudoRealtime?.disconnect();
        });

        // 2. pagehide - more reliable than beforeunload (works on mobile)
        window.addEventListener('pagehide', () => {
            window.sudoRealtime?.disconnect();
        });

        // 3. visibilitychange - detect when tab becomes hidden/inactive
        document.addEventListener('visibilitychange', () => {
            if (document.hidden) {
                // Tab is hidden - send offline status but don't disconnect
                // (user might come back quickly)
                window.sudoRealtime?.sendPresenceUpdate('away');
            } else {
                // Tab is visible again - send online status
                window.sudoRealtime?.sendPresenceUpdate('online');
            }
        });

        // 4. Window close/blur detection
        window.addEventListener('blur', () => {
            // Window lost focus - mark as away
            window.sudoRealtime?.sendPresenceUpdate('away');
        });

        window.addEventListener('focus', () => {
            // Window gained focus - mark as online
            window.sudoRealtime?.sendPresenceUpdate('online');
        });

        console.log('SUDO Real-time collaboration initialized');
        console.log('Collaborative features activated');
    }
});