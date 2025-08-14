// SUDO Kanban Board JavaScript
document.addEventListener('DOMContentLoaded', function() {
    initializeDragAndDrop();
    initializeModals();
});

// Drag and Drop functionality
function initializeDragAndDrop() {
    const columns = document.querySelectorAll('[data-sortable="tasks"]');
    
    columns.forEach(column => {
        new Sortable(column, {
            group: 'shared',
            animation: 150,
            ghostClass: 'opacity-50',
            chosenClass: 'transform scale-105',
            dragClass: 'transform rotate-2',
            onEnd: function(evt) {
                const taskId = evt.item.dataset.taskId;
                const newColumnId = evt.to.closest('[data-column-id]').dataset.columnId;
                const newPosition = evt.newIndex;
                
                // Send update to server
                fetch('/tasks/move', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded',
                    },
                    body: new URLSearchParams({
                        task_id: taskId,
                        column_id: newColumnId,
                        position: newPosition
                    })
                }).catch(error => {
                    console.error('Error moving task:', error);
                    // Revert the move on error
                    evt.from.insertBefore(evt.item, evt.from.children[evt.oldIndex]);
                });
            }
        });
    });
}

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
    const form = document.getElementById(`add-task-form-${columnId}`);
    if (form) {
        form.classList.add('hidden');
        form.reset();
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

// HTMX event handlers
document.body.addEventListener('htmx:afterRequest', function(evt) {
    // Re-initialize drag and drop after HTMX updates
    if (evt.detail.xhr.status === 200) {
        setTimeout(() => {
            initializeDragAndDrop();
            // Update task counts
            document.querySelectorAll('[data-column-id]').forEach(column => {
                updateTaskCount(column.dataset.columnId);
            });
        }, 100);
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
        }
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
        }
    }).then(response => {
        if (response.ok) {
            return response.text();
        }
        throw new Error('Failed to load task details');
    }).then(html => {
        modalContent.innerHTML = html;
    }).catch(error => {
        console.error('Error loading task details:', error);
        modalContent.innerHTML = '<p class="text-red-600">Failed to load task details</p>';
    });
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