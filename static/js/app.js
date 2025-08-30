// SUDO Kanban Board JavaScript
document.addEventListener('DOMContentLoaded', function() {
    console.log('DOM loaded, initializing application...');
    console.log('SortableJS available:', typeof Sortable !== 'undefined');
    initializeModals();
    
    // Force trigger htmx.onLoad for initial page load
    console.log('Manually triggering htmx.onLoad for initial load...');
    htmx.onLoad(document.body);
});

// HTMX + SortableJS Integration - Enhanced for Full Column Dropzone
htmx.onLoad(function(content) {
    console.log('HTMX onLoad triggered, initializing sortables...');
    console.log('Content element:', content);
    console.log('SortableJS available in onLoad:', typeof Sortable !== 'undefined');
    
    var sortables = content.querySelectorAll('[data-sortable="tasks"]');
    console.log('Found sortables:', sortables.length);
    console.log('Sortable elements:', sortables);
    
    // Also check if we can find task cards
    var taskCards = content.querySelectorAll('[data-task-id]');
    console.log('Found task cards:', taskCards.length);
    
    for (var i = 0; i < sortables.length; i++) {
        var sortable = sortables[i];
        console.log('Initializing sortable:', sortable.dataset.columnId);
        
        // Destroy existing instance if it exists
        if (sortable.sortableInstance) {
            sortable.sortableInstance.destroy();
            sortable.sortableInstance = null;
        }
        
        // Ensure the sortable container expands to full column height
        sortable.style.minHeight = '300px';
        sortable.style.transition = 'all 0.2s ease';
        sortable.style.flex = '1';
        
        var sortableInstance = new Sortable(sortable, {
            group: {
                name: 'shared',
                pull: true,
                put: true
            },
            animation: 100,
            delay: 0,
            delayOnTouchStart: false,
            delayOnTouchOnly: false,
            touchStartThreshold: 0,
            ghostClass: 'sortable-ghost',
            chosenClass: 'sortable-chosen', 
            dragClass: 'sortable-drag',
            fallbackOnBody: true,
            swapThreshold: 0.5,
            emptyInsertThreshold: 15,
            dragoverBubble: false,
            forceFallback: false,
            sort: true,
            onStart: function(evt) {
                console.log('Drag started for task:', evt.item.dataset.taskId);
                console.log('From column:', evt.from.dataset.columnId);
                console.log('Item element:', evt.item);
                
                // Store original parent for potential revert
                evt.item.originalParent = evt.from;
                evt.item.originalIndex = evt.oldIndex;
                
                // Add enhanced visual feedback to all columns
                document.querySelectorAll('[data-sortable="tasks"]').forEach(col => {
                    col.classList.add('drag-active');
                });
                
                // Add body class to prevent text selection
                document.body.classList.add('dragging');
                document.body.style.userSelect = 'none';
            },
            onMove: function(evt, originalEvent) {
                // Allow dropping anywhere in any column
                console.log('Move event - from:', evt.from.dataset.columnId, 'to:', evt.to.dataset.columnId);
                return true;
            },
            onChange: function(evt) {
                console.log('Change event - item moved within or between containers');
            },
            onEnd: function(evt) {
                console.log('Drag ended');
                console.log('Event details:', {
                    item: evt.item,
                    from: evt.from,
                    to: evt.to,
                    oldIndex: evt.oldIndex,
                    newIndex: evt.newIndex
                });
                
                // Remove visual feedback
                document.querySelectorAll('[data-sortable="tasks"]').forEach(col => {
                    col.classList.remove('drag-active');
                });
                
                // Remove body dragging class
                document.body.classList.remove('dragging');
                document.body.style.userSelect = '';
                
                var taskId = evt.item.dataset.taskId;
                var oldColumnId = evt.from.dataset.columnId || evt.from.closest('[data-column-id]').dataset.columnId;
                var newColumnId = evt.to.dataset.columnId || evt.to.closest('[data-column-id]').dataset.columnId;
                var newPosition = evt.newIndex;
                
                console.log('Task movement details:', {
                    taskId: taskId,
                    oldColumnId: oldColumnId,
                    newColumnId: newColumnId,
                    oldIndex: evt.oldIndex,
                    newIndex: newPosition,
                    actuallyMoved: !(oldColumnId === newColumnId && evt.oldIndex === evt.newIndex)
                });
                
                // Don't make API call if task wasn't actually moved
                if (oldColumnId === newColumnId && evt.oldIndex === evt.newIndex) {
                    console.log('Task position unchanged, skipping API call');
                    return;
                }
                
                // Emit custom event for task move
                document.dispatchEvent(new CustomEvent('taskMoved', {
                    detail: {
                        taskId: taskId,
                        oldColumnId: oldColumnId,
                        newColumnId: newColumnId,
                        oldPosition: evt.oldIndex,
                        newPosition: newPosition
                    }
                }));
                
                console.log('Sending task move request...');
                fetch('/tasks/move', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded',
                    },
                    credentials: 'include',
                    body: new URLSearchParams({
                        task_id: taskId,
                        column_id: newColumnId,
                        position: newPosition
                    })
                }).then(response => {
                    console.log('Server response status:', response.status);
                    if (!response.ok) {
                        throw new Error(`Server responded with status ${response.status}`);
                    }
                    return response.json();
                }).then(data => {
                    console.log('Task move successful:', data);
                    
                    // Update empty states for both columns
                    updateEmptyState(oldColumnId);
                    updateEmptyState(newColumnId);
                    
                    // Emit success event
                    document.dispatchEvent(new CustomEvent('taskMoveSuccess', {
                        detail: {
                            taskId: taskId,
                            newColumnId: newColumnId,
                            newPosition: newPosition,
                            data: data
                        }
                    }));
                }).catch(error => {
                    console.error('Error moving task:', error);
                    
                    // Emit error event
                    document.dispatchEvent(new CustomEvent('taskMoveError', {
                        detail: {
                            taskId: taskId,
                            error: error.message
                        }
                    }));
                    
                    // Revert the move on error
                    console.log('Reverting task move due to error');
                    if (evt.item.originalParent && evt.item.originalIndex !== undefined) {
                        if (evt.item.originalParent.children[evt.item.originalIndex]) {
                            evt.item.originalParent.insertBefore(evt.item, evt.item.originalParent.children[evt.item.originalIndex]);
                        } else {
                            evt.item.originalParent.appendChild(evt.item);
                        }
                    }
                });
            }
        });
        
        // Store instance for cleanup
        sortable.sortableInstance = sortableInstance;
    }
});

// Global function to manually reinitialize drag and drop for debugging
window.reinitializeDragAndDrop = function() {
    console.log('Manual drag and drop reinitialization requested');
    // Trigger HTMX onLoad for the entire document
    htmx.onLoad(document.body);
};

// Debug function to check sortable instances
window.debugSortables = function() {
    console.log('=== SORTABLE DEBUG INFO ===');
    const sortables = document.querySelectorAll('[data-sortable="tasks"]');
    console.log('Found sortable containers:', sortables.length);
    
    sortables.forEach((sortable, index) => {
        console.log(`Sortable ${index + 1}:`, {
            element: sortable,
            columnId: sortable.dataset.columnId,
            hasInstance: !!sortable.sortableInstance,
            children: sortable.children.length,
            childElements: Array.from(sortable.children).map(child => ({
                tagName: child.tagName,
                taskId: child.dataset.taskId,
                classes: child.className
            }))
        });
    });
    
    console.log('=== TASK CARDS ===');
    const taskCards = document.querySelectorAll('[data-task-id]');
    console.log('Found task cards:', taskCards.length);
    taskCards.forEach((card, index) => {
        console.log(`Task ${index + 1}:`, {
            taskId: card.dataset.taskId,
            parentColumn: card.closest('[data-sortable="tasks"]')?.dataset?.columnId,
            draggable: card.draggable,
            classes: card.className
        });
    });
    
    return {
        sortables: sortables.length,
        taskCards: taskCards.length,
        instances: Array.from(sortables).map(s => !!s.sortableInstance)
    };
};


// Modal functionality
function initializeModals() {
    // Close modal when clicking outside
    document.addEventListener('click', function(e) {
        if (e.target.classList.contains('fixed') && e.target.classList.contains('inset-0')) {
            closeAllModals();
        }
    });
    
    // Close modal on Escape key
    document.addEventListener('keydown', function(e) {
        if (e.key === 'Escape') {
            closeAllModals();
        }
    });
}

function closeAllModals() {
    const modals = document.querySelectorAll('.fixed.inset-0');
    modals.forEach(modal => {
        modal.classList.add('hidden');
    });
    
    // Hide all add task forms
    const addTaskForms = document.querySelectorAll('[id^="add-task-form-"]');
    addTaskForms.forEach(form => {
        form.classList.add('hidden');
    });
}

// Add Task Form Functions
function showAddTaskForm(columnId) {
    // Hide all other forms first
    const allForms = document.querySelectorAll('[id^="add-task-form-"]');
    allForms.forEach(form => form.classList.add('hidden'));
    
    // Show the specific form
    const form = document.getElementById(`add-task-form-${columnId}`);
    if (form) {
        form.classList.remove('hidden');
        const titleInput = form.querySelector('input[name="title"]');
        if (titleInput) {
            titleInput.focus();
        }
    }
}

function hideAddTaskForm(columnId) {
    const formContainer = document.getElementById(`add-task-form-${columnId}`);
    if (formContainer) {
        formContainer.classList.add('hidden');
        const form = formContainer.querySelector('form');
        if (form) {
            // Reset form to clear all field values
            form.reset();
            // Clear any manually set values that reset() might miss
            const inputs = form.querySelectorAll('input[type="text"], input[type="datetime-local"], textarea, select');
            inputs.forEach(input => {
                if (input.type === 'text' || input.type === 'datetime-local' || input.tagName.toLowerCase() === 'textarea') {
                    input.value = '';
                } else if (input.tagName.toLowerCase() === 'select') {
                    input.selectedIndex = 1; // Reset to "Medium" (index 1)
                }
            });
        }
    }
}

// Task Modal Functions
function showTaskModal(taskId) {
    console.log('Opening task modal for:', taskId);
    // TODO: Implement task editing modal
    // For now, we'll just log the task ID
}

// Nested Board Functions
function createNestedBoard(taskId) {
    console.log('Creating nested board for task:', taskId);
    // TODO: Implement nested board creation
    // This would create a new board with the task as parent
}

// Update task count in column headers
function updateTaskCount(columnId) {
    const column = document.querySelector(`[data-column-id="${columnId}"]`);
    if (column) {
        const tasks = column.querySelectorAll('.task-card');
        const countElement = column.querySelector('.bg-gray-200.text-gray-700');
        if (countElement) {
            countElement.textContent = tasks.length;
        }
    }
}

// Update empty state dropzone based on task count
function updateEmptyState(columnId) {
    const tasksContainer = document.querySelector(`#tasks-${columnId}`);
    const emptyStateContainer = tasksContainer?.parentElement?.querySelector('.empty-state-dropzone');
    
    if (!tasksContainer || !emptyStateContainer) {
        console.log('Could not find containers for column:', columnId);
        return;
    }
    
    const taskCards = tasksContainer.querySelectorAll('.task-card');
    const hasNoTasks = taskCards.length === 0;
    
    console.log(`Updating empty state for column ${columnId}:`, {
        taskCount: taskCards.length,
        hasNoTasks: hasNoTasks
    });
    
    if (hasNoTasks) {
        // Show full empty state with "add new task" button
        emptyStateContainer.innerHTML = `
            <div class="flex items-center justify-center h-32 border-2 border-dashed border-gray-300 rounded-lg text-gray-500">
                <div class="text-center">
                    <svg class="w-8 h-8 mx-auto mb-2 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
                    </svg>
                    <p class="text-sm">Drop tasks here or</p>
                    <button 
                        onclick="showAddTaskForm('${columnId}')"
                        class="text-blue-600 hover:text-blue-800 text-sm font-medium"
                    >
                        add a new task
                    </button>
                </div>
            </div>
        `;
    } else {
        // Show minimal drop zone at bottom
        emptyStateContainer.innerHTML = `
            <div class="flex items-center justify-center h-16 border-2 border-dashed border-gray-200 rounded-lg text-gray-400 mt-3 hover:border-gray-300 transition-colors">
                <p class="text-sm">Drop tasks here</p>
            </div>
        `;
    }
}

// Update task card in DOM with new data
function updateTaskCardInDOM(taskId, taskData) {
    const taskCard = document.querySelector(`[data-task-id="${taskId}"]`);
    if (!taskCard) {
        console.log('Task card not found:', taskId);
        return;
    }
    
    console.log('Updating task card in DOM:', taskId, taskData);
    
    // Update title
    const titleElement = taskCard.querySelector('h4');
    if (titleElement && taskData.title) {
        titleElement.textContent = taskData.title;
    }
    
    // Update description
    const descriptionElement = taskCard.querySelector('p');
    if (descriptionElement && taskData.description !== undefined) {
        if (taskData.description) {
            descriptionElement.textContent = taskData.description;
            descriptionElement.style.display = 'block';
        } else {
            descriptionElement.style.display = 'none';
        }
    }
    
    // Update priority badge
    const priorityBadge = taskCard.querySelector('.priority-badge, span[class*="bg-"]');
    if (priorityBadge && taskData.priority) {
        priorityBadge.textContent = taskData.priority;
        priorityBadge.className = `text-xs font-medium px-2 py-1 rounded ${getPriorityBadgeClass(taskData.priority)}`;
    }
    
    // Update priority indicator dot
    const priorityDot = taskCard.querySelector('.w-2.h-2.rounded-full');
    if (priorityDot && taskData.priority) {
        priorityDot.className = `w-2 h-2 rounded-full ${getPriorityColorClass(taskData.priority)}`;
    }
    
    // Update completion status
    const completionButton = taskCard.querySelector('button[onclick*="toggleTaskComplete"]');
    if (completionButton && taskData.completed !== undefined) {
        if (taskData.completed) {
            completionButton.className = 'text-green-500 hover:text-green-600';
            completionButton.innerHTML = `
                <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"></path>
                </svg>
            `;
            completionButton.title = 'Mark as incomplete';
        } else {
            completionButton.className = 'text-gray-400 hover:text-gray-600';
            completionButton.innerHTML = `
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
                </svg>
            `;
            completionButton.title = 'Mark as complete';
        }
    }
    
    // Add update animation
    taskCard.style.transition = 'all 0.3s ease';
    taskCard.style.transform = 'scale(1.02)';
    setTimeout(() => {
        taskCard.style.transform = 'scale(1)';
    }, 200);
}

// Show notification to user
function showNotification(message, type = 'info') {
    // Remove existing notifications
    const existingNotifications = document.querySelectorAll('.notification');
    existingNotifications.forEach(n => n.remove());
    
    const notification = document.createElement('div');
    notification.className = `notification fixed top-4 right-4 px-4 py-3 rounded-lg shadow-lg z-50 transition-all duration-300 transform`;
    
    const bgColorClass = {
        'success': 'bg-green-100 border border-green-400 text-green-700',
        'error': 'bg-red-100 border border-red-400 text-red-700',
        'warning': 'bg-yellow-100 border border-yellow-400 text-yellow-700',
        'info': 'bg-blue-100 border border-blue-400 text-blue-700'
    };
    
    notification.className += ` ${bgColorClass[type] || bgColorClass.info}`;
    
    const icon = {
        'success': '✓',
        'error': '✕',
        'warning': '⚠',
        'info': 'ℹ'
    };
    
    notification.innerHTML = `
        <div class="flex items-center">
            <span class="mr-2 font-bold">${icon[type] || icon.info}</span>
            <span>${message}</span>
        </div>
    `;
    
    document.body.appendChild(notification);
    
    // Animate in
    setTimeout(() => {
        notification.style.opacity = '1';
        notification.style.transform = 'translateY(0)';
    }, 100);
    
    // Auto-remove after 3 seconds
    setTimeout(() => {
        notification.style.opacity = '0';
        notification.style.transform = 'translateY(-20px)';
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}

// Helper functions for priority styling
function getPriorityColorClass(priority) {
    const colors = {
        'Urgent': 'bg-red-500',
        'High': 'bg-orange-500', 
        'Medium': 'bg-yellow-500',
        'Low': 'bg-green-500'
    };
    return colors[priority] || 'bg-gray-500';
}

function getPriorityBadgeClass(priority) {
    const classes = {
        'Urgent': 'bg-red-100 text-red-800',
        'High': 'bg-orange-100 text-orange-800',
        'Medium': 'bg-yellow-100 text-yellow-800',
        'Low': 'bg-green-100 text-green-800'
    };
    return classes[priority] || 'bg-gray-100 text-gray-800';
}

// Real-time updates (placeholder for WebSocket implementation)
function initializeRealTimeUpdates() {
    // TODO: Implement WebSocket connection for real-time collaboration
    console.log('Real-time updates would be initialized here');
}

// Form validation helpers
function validateEmail(email) {
    const re = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return re.test(email);
}

function validateTaskForm(form) {
    const title = form.querySelector('input[name="title"]').value.trim();
    if (!title) {
        alert('Task title is required');
        return false;
    }
    return true;
}

// Priority color helpers
function getPriorityColor(priority) {
    const colors = {
        'Low': 'bg-green-100 text-green-800',
        'Medium': 'bg-yellow-100 text-yellow-800', 
        'High': 'bg-red-100 text-red-800',
        'Urgent': 'bg-red-200 text-red-900'
    };
    return colors[priority] || colors['Medium'];
}

// Keyboard shortcuts
document.addEventListener('keydown', function(e) {
    // Ctrl/Cmd + K to quickly add task to first column
    if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault();
        const firstColumn = document.querySelector('[data-column-id]');
        if (firstColumn) {
            const columnId = firstColumn.dataset.columnId;
            showAddTaskForm(columnId);
        }
    }
    
    // Escape to close modals/forms
    if (e.key === 'Escape') {
        closeAllModals();
    }
});

// Auto-save draft functionality (localStorage backup)
function saveDraft(formId, data) {
    try {
        localStorage.setItem(`draft_${formId}`, JSON.stringify(data));
    } catch (error) {
        console.warn('Could not save draft:', error);
    }
}

function loadDraft(formId) {
    try {
        const draft = localStorage.getItem(`draft_${formId}`);
        return draft ? JSON.parse(draft) : null;
    } catch (error) {
        console.warn('Could not load draft:', error);
        return null;
    }
}

function clearDraft(formId) {
    try {
        localStorage.removeItem(`draft_${formId}`);
    } catch (error) {
        console.warn('Could not clear draft:', error);
    }
}

// Initialize tooltips (if using a tooltip library)
function initializeTooltips() {
    // TODO: Add tooltip initialization if needed
}

// Performance optimization: debounce function
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// Update task counts after HTMX operations
document.body.addEventListener('htmx:afterRequest', function(evt) {
    if (evt.detail.xhr.status === 200) {
        // Update task counts and empty states
        document.querySelectorAll('[data-column-id]').forEach(column => {
            const columnId = column.dataset.columnId;
            updateTaskCount(columnId);
            updateEmptyState(columnId);
        });
    }
});

document.body.addEventListener('htmx:beforeRequest', function(evt) {
    // Add loading states
    const trigger = evt.detail.elt;
    if (trigger.tagName === 'FORM') {
        const submitBtn = trigger.querySelector('button[type="submit"]');
        if (submitBtn) {
            submitBtn.disabled = true;
            submitBtn.textContent = 'Loading...';
        }
    }
});

document.body.addEventListener('htmx:afterRequest', function(evt) {
    // Remove loading states
    const trigger = evt.detail.elt;
    if (trigger.tagName === 'FORM') {
        const submitBtn = trigger.querySelector('button[type="submit"]');
        if (submitBtn) {
            submitBtn.disabled = false;
            // Restore original text based on form context
            if (trigger.querySelector('input[name="title"]')) {
                submitBtn.textContent = 'Add Task';
            } else if (trigger.querySelector('input[name="email"]')) {
                submitBtn.textContent = 'Send Invite';
            } else {
                submitBtn.textContent = 'Submit';
            }
        }
    }
});

// Column Menu Functions
function toggleColumnMenu(columnId) {
    // Hide all other column menus first
    const allMenus = document.querySelectorAll('[id^="column-menu-"]');
    allMenus.forEach(menu => {
        if (menu.id !== `column-menu-${columnId}`) {
            menu.classList.add('hidden');
        }
    });
    
    // Toggle the specific menu
    const menu = document.getElementById(`column-menu-${columnId}`);
    if (menu) {
        menu.classList.toggle('hidden');
    }
}

// Task Modal Functions
function openTaskModal(columnId) {
    const modal = document.getElementById('task-modal');
    if (modal) {
        modal.classList.remove('hidden');
        // Store the column ID for form submission
        modal.dataset.columnId = columnId;
    }
}

// Task completion toggle
function toggleTaskComplete(taskId) {
    const taskCard = document.querySelector(`[data-task-id="${taskId}"]`);
    if (!taskCard) return;
    
    // Get current completion status from UI
    const isCompleted = taskCard.querySelector('.text-green-500') !== null;
    const endpoint = isCompleted ? `/api/tasks/${taskId}/reopen` : `/api/tasks/${taskId}/complete`;
    
    fetch(endpoint, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/x-www-form-urlencoded',
        },
        credentials: 'include'
    }).then(response => {
        if (response.ok) {
            // Refresh the board or update UI
            location.reload();
        } else {
            console.error('Failed to toggle task completion');
        }
    }).catch(error => {
        console.error('Error toggling task completion:', error);
    });
}

// Open task details modal
function openTaskDetails(taskId) {
    const modal = document.getElementById('task-modal');
    const modalContent = document.getElementById('task-modal-content');
    
    if (!modal || !modalContent) return;
    
    // Store task ID on modal for later use
    modal.dataset.taskId = taskId;
    
    // Show modal with loading state
    modal.classList.remove('hidden');
    modalContent.innerHTML = `
        <div class="animate-pulse space-y-4">
            <div class="h-4 bg-gray-200 rounded w-3/4"></div>
            <div class="h-4 bg-gray-200 rounded w-1/2"></div>
            <div class="h-20 bg-gray-200 rounded"></div>
        </div>
    `;
    
    // Fetch task details
    fetch(`/tasks/${taskId}`, {
        method: 'GET',
        headers: {
            'HX-Request': 'true'
        },
        credentials: 'include'
    }).then(response => {
        if (response.ok) {
            return response.text();
        }
        throw new Error('Failed to load task details');
    }).then(html => {
        modalContent.innerHTML = html;
        
        // Add event delegation for task action buttons
        setupTaskActionButtons();
    }).catch(error => {
        console.error('Error loading task details:', error);
        modalContent.innerHTML = '<p class="text-red-600">Failed to load task details</p>';
    });
}

// Setup event delegation for task action buttons
function setupTaskActionButtons() {
    const modal = document.getElementById('task-modal');
    if (!modal) return;
    
    // Remove existing listeners to avoid duplicates
    modal.removeEventListener('click', handleTaskActionClick);
    
    // Add event delegation for task action buttons
    modal.addEventListener('click', handleTaskActionClick);
}

// Handle clicks on task action buttons
function handleTaskActionClick(event) {
    const button = event.target.closest('.task-action-btn');
    if (!button) return;
    
    const taskId = button.dataset.taskId;
    const action = button.dataset.action;
    
    if (!taskId || !action) return;
    
    event.preventDefault();
    
    switch (action) {
        case 'delete':
            deleteTask(taskId);
            break;
        case 'save-changes':
            saveTaskChanges(taskId);
            break;
        case 'convert-to-subboard':
            convertToSubBoard(taskId);
            break;
    }
}

// Board Menu Functions (for dashboard)
function toggleBoardMenu(button) {
    // Close all other menus
    document.querySelectorAll('.board-menu').forEach(menu => {
        if (menu !== button.parentElement.querySelector('.board-menu')) {
            menu.classList.add('hidden');
        }
    });
    
    // Toggle this menu
    const menu = button.parentElement.querySelector('.board-menu');
    if (menu) {
        menu.classList.toggle('hidden');
    }
}

// Close menus when clicking outside
document.addEventListener('click', function(event) {
    if (!event.target.closest('.board-menu') && !event.target.closest('button')) {
        document.querySelectorAll('.board-menu').forEach(menu => {
            menu.classList.add('hidden');
        });
    }
});

// Delete Functions
function deleteTask(taskId) {
    if (confirm('Are you sure you want to delete this task? This action cannot be undone.')) {
        console.log('Deleting task:', taskId);
        
        // Close modal if it's open
        const modal = document.getElementById('task-modal');
        if (modal && !modal.classList.contains('hidden')) {
            closeTaskModal();
        }
        
        // Find the task card to get column info before deletion
        const taskCard = document.querySelector(`[data-task-id="${taskId}"]`);
        const columnId = taskCard?.closest('[data-sortable="tasks"]')?.dataset?.columnId;
        
        fetch(`/tasks/${taskId}`, {
            method: 'DELETE',
            headers: {
                'Content-Type': 'application/json',
            },
            credentials: 'include'
        }).then(response => {
            if (response.ok) {
                console.log('Task deleted successfully');
                
                // Remove the task card from DOM with animation
                if (taskCard) {
                    taskCard.style.transition = 'all 0.3s ease';
                    taskCard.style.opacity = '0';
                    taskCard.style.transform = 'scale(0.8)';
                    
                    setTimeout(() => {
                        taskCard.remove();
                        
                        // Update task counts and empty states
                        if (columnId) {
                            updateTaskCount(columnId);
                            updateEmptyState(columnId);
                        }
                        
                        // Update all column task counts
                        document.querySelectorAll('[data-column-id]').forEach(column => {
                            const colId = column.dataset.columnId;
                            updateTaskCount(colId);
                            updateEmptyState(colId);
                        });
                    }, 300);
                }
                
                // Show success notification
                showNotification('Task deleted successfully!', 'success');
                
            } else {
                throw new Error(`Failed to delete task: ${response.status}`);
            }
        }).catch(error => {
            console.error('Error deleting task:', error);
            showNotification('Failed to delete task. Please try again.', 'error');
        });
    }
}

function deleteColumn(columnId) {
    const column = document.querySelector(`[data-column-id="${columnId}"]`);
    const taskCount = column ? column.querySelectorAll('[data-task-id]').length : 0;
    
    let confirmMessage = 'Are you sure you want to delete this column?';
    if (taskCount > 0) {
        confirmMessage = `Are you sure you want to delete this column and all ${taskCount} tasks in it? This action cannot be undone.`;
    }
    
    if (confirm(confirmMessage)) {
        fetch(`/columns/${columnId}`, {
            method: 'DELETE',
            headers: {
                'Content-Type': 'application/json',
            },
            credentials: 'include'
        }).then(response => {
            if (response.ok) {
                // Remove the column from DOM
                if (column) {
                    column.remove();
                }
            } else {
                alert('Failed to delete column. Please try again.');
            }
        }).catch(error => {
            console.error('Error deleting column:', error);
            alert('Failed to delete column. Please try again.');
        });
    }
}

function deleteBoard(boardId) {
    console.log('deleteBoard called with boardId:', boardId);
    console.log('boardId type:', typeof boardId);
    console.log('boardId length:', boardId ? boardId.length : 'undefined');
    
    if (confirm('Are you sure you want to delete this entire board? This will permanently delete all columns and tasks. This action cannot be undone.')) {
        console.log('User confirmed deletion, making request to:', `/boards/${boardId}`);
        
        // Check if we have a valid session
        console.log('Document cookies:', document.cookie);
        
        const requestUrl = `/boards/${boardId}`;
        console.log('Full request URL:', window.location.origin + requestUrl);
        
        fetch(requestUrl, {
            method: 'DELETE',
            headers: {
                'Content-Type': 'application/json',
            },
            credentials: 'include'
        }).then(response => {
            console.log('Response received!');
            console.log('Response status:', response.status);
            console.log('Response ok:', response.ok);
            console.log('Response headers:', [...response.headers.entries()]);
            
            if (response.ok) {
                // Redirect to dashboard
                console.log('Deletion successful, redirecting to dashboard');
                window.location.href = '/dashboard';
            } else {
                return response.text().then(text => {
                    console.error('Deletion failed with status:', response.status);
                    console.error('Deletion failed with response:', text);
                    alert(`Failed to delete board. Status: ${response.status}. Please try again.`);
                });
            }
        }).catch(error => {
            console.error('Network error details:');
            console.error('Error name:', error.name);
            console.error('Error message:', error.message);
            console.error('Error stack:', error.stack);
            console.error('Full error object:', error);
            alert('Failed to delete board. Network error. Please try again.');
        });
    } else {
        console.log('User cancelled deletion');
    }
}

// Close task modal
function closeTaskModal() {
    const modal = document.getElementById('task-modal');
    if (modal) {
        modal.classList.add('hidden');
    }
}

// Save task changes from modal
function saveTaskChanges(taskId) {
    const form = document.getElementById('task-modal');
    if (!form) {
        console.error('Task modal not found');
        return;
    }
    
    // If taskId is not provided, try to get it from the modal's dataset
    if (!taskId) {
        taskId = form.dataset.taskId;
        if (!taskId) {
            console.error('Task ID not found');
            return;
        }
    }

    // Show loading state on save button
    const saveButton = form.querySelector('button[onclick*="saveTaskChanges"]');
    const originalText = saveButton ? saveButton.textContent : 'Save Changes';
    if (saveButton) {
        saveButton.disabled = true;
        saveButton.textContent = 'Saving...';
    }

    // Gather form data
    const title = document.getElementById('task-title')?.value?.trim() || '';
    const description = document.getElementById('task-description')?.value?.trim() || '';
    const priority = document.getElementById('task-priority')?.value || '';
    const deadline = document.getElementById('task-deadline')?.value || '';
    const assigneeId = document.getElementById('task-assignee')?.value || '';
    const completed = document.getElementById('task-completed')?.checked || false;

    // Validation
    if (!title) {
        alert('Task title is required');
        if (saveButton) {
            saveButton.disabled = false;
            saveButton.textContent = originalText;
        }
        return;
    }

    // Build form data
    const formData = new URLSearchParams();
    formData.append('title', title);
    formData.append('description', description);
    formData.append('priority', priority);
    if (deadline) formData.append('deadline', deadline);
    if (assigneeId && assigneeId !== '') {
        formData.append('assignee_id', assigneeId);
    } else {
        formData.append('assignee_id', 'unassign');
    }
    formData.append('completed', completed.toString());

    console.log('Saving task changes:', { taskId, title, priority, completed });

    // Send update request
    fetch(`/tasks/${taskId}`, {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/x-www-form-urlencoded',
        },
        credentials: 'include',
        body: formData
    }).then(response => {
        if (response.ok) {
            console.log('Task updated successfully');
            closeTaskModal();
            
            // Update the task card in the DOM with new data
            updateTaskCardInDOM(taskId, {
                title,
                description,
                priority,
                completed,
                deadline: deadline || null
            });
            
            // Show success notification
            showNotification('Task updated successfully!', 'success');
            
        } else {
            throw new Error(`Failed to save task changes: ${response.status}`);
        }
    }).catch(error => {
        console.error('Error saving task changes:', error);
        showNotification('Failed to save task changes. Please try again.', 'error');
        
        // Restore button state
        if (saveButton) {
            saveButton.disabled = false;
            saveButton.textContent = originalText;
        }
    });
}

// Convert task to sub-board
function convertToSubBoard(taskId) {
    if (confirm('Convert this task to a sub-board? This will create a new board based on this task.')) {
        fetch(`/tasks/${taskId}/convert-to-board`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
            },
            credentials: 'include'
        }).then(response => {
            if (response.ok) {
                // The response should contain a redirect header
                const redirect = response.headers.get('HX-Redirect');
                if (redirect) {
                    window.location.href = redirect;
                } else {
                    location.reload();
                }
            } else {
                console.error('Failed to convert to sub-board');
                alert('Failed to convert to sub-board. Please try again.');
            }
        }).catch(error => {
            console.error('Error converting to sub-board:', error);
            alert('Failed to convert to sub-board. Please try again.');
        });
    }
}

// Copy task link to clipboard
function copyTaskLink() {
    const modal = document.getElementById('task-modal');
    const taskId = modal?.dataset?.taskId;
    if (!taskId) return;
    
    const taskUrl = `${window.location.origin}${window.location.pathname}#task-${taskId}`;
    
    navigator.clipboard.writeText(taskUrl).then(() => {
        // Show temporary feedback
        const button = event.target.closest('button');
        const originalText = button.innerHTML;
        button.innerHTML = '<svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path></svg> Copied!';
        setTimeout(() => {
            button.innerHTML = originalText;
        }, 2000);
    }).catch(error => {
        console.error('Failed to copy link:', error);
        alert('Failed to copy link to clipboard');
    });
}

// Dark Mode Functions
function toggleDarkMode() {
    const html = document.documentElement;
    const isDark = html.classList.contains('dark');
    
    if (isDark) {
        html.classList.remove('dark');
        localStorage.setItem('darkMode', 'false');
    } else {
        html.classList.add('dark');
        localStorage.setItem('darkMode', 'true');
    }
}

function initializeDarkMode() {
    const darkMode = localStorage.getItem('darkMode') === 'true' || 
                   (!localStorage.getItem('darkMode') && window.matchMedia('(prefers-color-scheme: dark)').matches);
    if (darkMode) {
        document.documentElement.classList.add('dark');
    }
}

// Initialize dark mode on page load
document.addEventListener('DOMContentLoaded', function() {
    initializeDarkMode();
});

// Export functions for global access
window.showAddTaskForm = showAddTaskForm;
window.hideAddTaskForm = hideAddTaskForm;
window.showTaskModal = showTaskModal;
window.openTaskModal = openTaskModal;
window.toggleColumnMenu = toggleColumnMenu;
window.toggleTaskComplete = toggleTaskComplete;
window.openTaskDetails = openTaskDetails;
window.createNestedBoard = createNestedBoard;
window.toggleBoardMenu = toggleBoardMenu;
window.deleteTask = deleteTask;
window.deleteColumn = deleteColumn;
window.deleteBoard = deleteBoard;
window.closeTaskModal = closeTaskModal;
window.saveTaskChanges = saveTaskChanges;
window.convertToSubBoard = convertToSubBoard;
window.copyTaskLink = copyTaskLink;
window.toggleDarkMode = toggleDarkMode;

// Export debug functions
window.debugSortables = debugSortables;
window.reinitializeDragAndDrop = reinitializeDragAndDrop;

// Enhanced Drag and Drop Event Handlers
document.addEventListener('taskMoved', function(e) {
    const { taskId, oldColumnId, newColumnId, oldPosition, newPosition } = e.detail;
    
    // Update task counts and empty states for both columns if they're different
    if (oldColumnId !== newColumnId) {
        updateTaskCount(oldColumnId);
        updateTaskCount(newColumnId);
        updateEmptyState(oldColumnId);
        updateEmptyState(newColumnId);
    }
    
    // Optional: Show user feedback
    console.log(`Task ${taskId} moved from column ${oldColumnId} to ${newColumnId}`);
});

document.addEventListener('taskMoveSuccess', function(e) {
    const { taskId, newColumnId, newPosition } = e.detail;
    
    // Optional: Show success notification or update UI
    console.log(`Task ${taskId} successfully moved to column ${newColumnId} at position ${newPosition}`);
    
    // Update task counts to ensure accuracy
    document.querySelectorAll('[data-column-id]').forEach(column => {
        updateTaskCount(column.dataset.columnId);
    });
});

document.addEventListener('taskMoveError', function(e) {
    const { taskId, error } = e.detail;
    
    // Show error notification to user
    console.error(`Failed to move task ${taskId}:`, error);
    
    // Optional: Show user-friendly error message
    const errorDiv = document.createElement('div');
    errorDiv.className = 'fixed top-4 right-4 bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded z-50';
    errorDiv.innerHTML = `
        <div class="flex items-center">
            <svg class="w-4 h-4 mr-2" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"></path>
            </svg>
            <span>Failed to move task. Please try again.</span>
        </div>
    `;
    
    document.body.appendChild(errorDiv);
    
    // Auto-remove error message after 3 seconds
    setTimeout(() => {
        if (errorDiv.parentNode) {
            document.body.removeChild(errorDiv);
        }
    }, 3000);
});