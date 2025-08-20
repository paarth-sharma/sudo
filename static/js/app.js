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
                    credentials: 'include',
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

// Delete Functions
function deleteTask(taskId) {
    if (confirm('Are you sure you want to delete this task? This action cannot be undone.')) {
        fetch(`/tasks/${taskId}`, {
            method: 'DELETE',
            headers: {
                'Content-Type': 'application/json',
            },
            credentials: 'include'
        }).then(response => {
            if (response.ok) {
                // Remove the task card from DOM
                const taskCard = document.querySelector(`[data-task-id="${taskId}"]`);
                if (taskCard) {
                    taskCard.remove();
                    // Update task counts in column headers
                    document.querySelectorAll('[data-column-id]').forEach(column => {
                        updateTaskCount(column.dataset.columnId);
                    });
                }
            } else {
                alert('Failed to delete task. Please try again.');
            }
        }).catch(error => {
            console.error('Error deleting task:', error);
            alert('Failed to delete task. Please try again.');
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
    if (!form) return;

    // Gather form data
    const title = document.getElementById('task-title')?.value || '';
    const description = document.getElementById('task-description')?.value || '';
    const priority = document.getElementById('task-priority')?.value || '';
    const deadline = document.getElementById('task-deadline')?.value || '';
    const assigneeId = document.getElementById('task-assignee')?.value || '';
    const completed = document.getElementById('task-completed')?.checked || false;

    // Build form data
    const formData = new URLSearchParams();
    if (title) formData.append('title', title);
    if (description) formData.append('description', description);
    if (priority) formData.append('priority', priority);
    if (deadline) formData.append('deadline', deadline);
    if (assigneeId) formData.append('assignee_id', assigneeId === '' ? 'unassign' : assigneeId);
    formData.append('completed', completed);

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
            // Close modal and refresh the page to show changes
            closeTaskModal();
            location.reload();
        } else {
            console.error('Failed to save task changes');
            alert('Failed to save task changes. Please try again.');
        }
    }).catch(error => {
        console.error('Error saving task changes:', error);
        alert('Failed to save task changes. Please try again.');
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
    const taskId = document.getElementById('task-modal')?.dataset?.taskId;
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