// SUDO Kanban Board JavaScript
document.addEventListener('DOMContentLoaded', function() {
    console.log('DOM loaded, initializing application...');
    console.log('SortableJS available:', typeof Sortable !== 'undefined');
    initializeModals();
    
    // Add HTMX error debugging
    document.body.addEventListener('htmx:responseError', function(evt) {
        console.error('HTMX Response Error:', evt.detail);
        console.error('Status:', evt.detail.xhr.status);
        console.error('Response:', evt.detail.xhr.responseText);
    });

    document.body.addEventListener('htmx:sendError', function(evt) {
        console.error('HTMX Send Error:', evt.detail);
    });

    // Listen for board member removals (DELETE requests to members endpoint)
    document.body.addEventListener('htmx:beforeRequest', function(evt) {
        const requestPath = evt.detail.pathInfo?.requestPath || '';
        const verb = evt.detail.requestConfig?.verb || '';

        // If deleting a board member, refresh collaborators count after success
        if (verb === 'DELETE' && requestPath.includes('/members/')) {
            console.log('Board member deletion detected');
            // Set a flag to refresh after the request completes
            evt.detail.elt.dataset.refreshCollaborators = 'true';
        }
    });

    document.body.addEventListener('htmx:afterRequest', function(evt) {
        // Check if we flagged this request to refresh collaborators
        if (evt.detail.elt.dataset.refreshCollaborators === 'true' && evt.detail.successful) {
            console.log('Board member deleted, refreshing collaborators count');
            delete evt.detail.elt.dataset.refreshCollaborators;
            refreshCollaboratorsCount();
        }
    });
    
    document.body.addEventListener('htmx:targetError', function(evt) {
        console.error('HTMX Target Error Details:', {
            target: evt.detail.target,
            element: evt.detail.elt,
            xhr: evt.detail.xhr,
            requestConfig: evt.detail.requestConfig
        });
    });

    // Listen for board creation to update search modal
    document.body.addEventListener('htmx:afterSwap', function(evt) {
        const requestPath = evt.detail.pathInfo?.requestPath || '';

        // Handle board creation
        if (requestPath === '/boards' && evt.detail.target?.id === 'boards-grid') {
            // A new board was created and added to the grid
            // Extract board data from the newly added card
            const newBoardCard = evt.detail.target.lastElementChild;
            if (newBoardCard) {
                const boardId = newBoardCard.getAttribute('data-board-id');
                const boardTitle = newBoardCard.querySelector('h3 a')?.textContent?.trim();
                const boardDescription = newBoardCard.querySelector('p')?.textContent?.trim() || '';
                const boardLink = newBoardCard.querySelector('h3 a')?.getAttribute('href');

                if (boardId && boardTitle && boardLink) {
                    // Add to search modal's recent boards
                    addBoardToSearchModal(boardId, boardTitle, boardDescription, boardLink);

                    // Update the main boards count
                    updateMainBoardsCount(1);

                    // Update the total boards stat
                    updateTotalBoardsCount(1);
                }
            }
        }

        // Handle board member additions (invite)
        if (requestPath && (requestPath.includes('/invite') || requestPath.includes('/members'))) {
            console.log('Board member change detected, refreshing collaborators count');
            // Refresh collaborators count from server
            refreshCollaboratorsCount();
        }

        // Handle task creation (when a task is added to a column)
        if (requestPath && requestPath.includes('/tasks') && evt.detail.target?.hasAttribute('data-sortable')) {
            // A new task was added to a column
            const targetElement = evt.detail.target;
            const newTaskCard = targetElement.lastElementChild;

            if (newTaskCard && newTaskCard.hasAttribute('data-task-id')) {
                // Check if it's an active (not completed) task
                const isActiveTask = newTaskCard.querySelector('.text-green-500') === null;

                if (isActiveTask) {
                    // Increment active tasks count
                    updateActiveTasksCount(1);
                }

                // Check if task is due soon
                const deadline = getTaskDeadline(newTaskCard);
                if (deadline) {
                    // Use simplified check if we got 'soon' flag, otherwise use date check
                    if (deadline === 'soon' || isTaskDueSoon(deadline)) {
                        updateDueSoonCount(1);
                    }
                }

                console.log('New task created, stats updated');
            }
        }
    });

    // Periodic sync for collaborators count (every 60 seconds) as a safety net
    // Only run on dashboard pages where stats are visible
    if (document.querySelector('[data-stat="collaborators"]')) {
        setInterval(function() {
            refreshCollaboratorsCount();
            console.log('Periodic collaborators count sync triggered');
        }, 60000); // Every 60 seconds
    }

    // Check for tasks and update sub-board button visibility
    setTimeout(checkTasksAndUpdateButton, 500);

    // Force trigger initialization for initial page load
    console.log('Manually triggering sortables initialization for initial load...');
    if (typeof initializeSortables === 'function') {
        initializeSortables(document.body);
    }
    
    // Set up a watchdog to periodically check and re-enable sortables if needed
    setInterval(function() {
        const disabledSortables = [];
        document.querySelectorAll('[data-sortable="tasks"]').forEach(container => {
            if (container.sortableInstance && container.sortableInstance.option('disabled')) {
                disabledSortables.push(container.dataset.columnId);
                container.sortableInstance.option("disabled", false);
            }
        });
        
        if (disabledSortables.length > 0) {
            console.log('Watchdog re-enabled sortables for columns:', disabledSortables);
        }
    }, 5000); // Check every 5 seconds
});

// HTMX + SortableJS Integration - Enhanced for Full Column Dropzone
function initializeSortables(content) {
    console.log('HTMX onLoad triggered, initializing sortables...');
    console.log('Content element:', content);
    console.log('SortableJS available in onLoad:', typeof Sortable !== 'undefined');
    
    // Use document.querySelectorAll instead of content.querySelectorAll to fix HTMX integration
    var sortables = document.querySelectorAll('[data-sortable="tasks"]');
    console.log('Found sortables:', sortables.length);
    console.log('Sortable elements:', sortables);
    
    // Also check if we can find task cards (use document for consistency)
    var taskCards = document.querySelectorAll('[data-task-id]');
    console.log('Found task cards:', taskCards.length);
    
    for (var i = 0; i < sortables.length; i++) {
        var sortable = sortables[i];
        console.log('Initializing sortable:', sortable.dataset.columnId);
        
        // Destroy existing instance if it exists to prevent duplicates
        if (sortable.sortableInstance) {
            console.log('Destroying existing sortable instance for column:', sortable.dataset.columnId);
            sortable.sortableInstance.destroy();
            sortable.sortableInstance = null;
        }
        
        // Let CSS and height equalization system handle the styling
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
                
                // Disable sortable during HTMX request to prevent conflicts
                this.option("disabled", true);
                
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
                
                const sortableInstance = this;
                
                // Add timeout to prevent hanging requests
                const controller = new AbortController();
                const timeoutId = setTimeout(() => controller.abort(), 10000); // 10 second timeout
                
                fetch('/tasks/move', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded',
                    },
                    credentials: 'include',
                    signal: controller.signal,
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
                    clearTimeout(timeoutId); // Clear timeout on success
                    console.log('Task move successful:', data);
                    
                    // Update task counts and empty states for both columns
                    updateTaskCount(oldColumnId);
                    updateTaskCount(newColumnId);
                    updateEmptyState(oldColumnId);
                    updateEmptyState(newColumnId);
                    
                    // Trigger height equalization immediately for real-time column resizing
                    console.log('Triggering immediate height equalization after task move success');
                    equalizeColumnHeightsOptimized();
                    
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
                    clearTimeout(timeoutId); // Clear timeout on error
                    console.error('Error moving task:', error);
                    
                    // Handle timeout specifically
                    if (error.name === 'AbortError') {
                        console.error('Task move request timed out after 10 seconds');
                        showNotification('Request timed out. Please try again.', 'error');
                        return;
                    }
                    
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
                }).finally(() => {
                    // Always re-enable sortable after request completes (success or error)
                    console.log('Re-enabling sortable after task move request');
                    if (sortableInstance && typeof sortableInstance.option === 'function') {
                        sortableInstance.option("disabled", false);
                    }
                    
                    // Height equalization is now handled in success handler for better timing
                    
                    // Also re-enable all other sortables to be safe
                    document.querySelectorAll('[data-sortable="tasks"]').forEach(container => {
                        if (container.sortableInstance && typeof container.sortableInstance.option === 'function') {
                            container.sortableInstance.option("disabled", false);
                        }
                    });
                });
            }
        });
        
        // Store instance for cleanup
        sortable.sortableInstance = sortableInstance;
        
        // Add HTMX event listener to re-enable sortable after swaps
        sortable.addEventListener("htmx:afterSwap", function() {
            // Equalize column heights after HTMX content swap
            if (typeof equalizeColumnHeightsDebounced === 'function') {
                equalizeColumnHeightsDebounced(250);
            }
            console.log('HTMX swap completed, re-enabling sortable for column:', sortable.dataset.columnId);
            if (sortableInstance) {
                sortableInstance.option("disabled", false);
            }
        });
    }
}

// Safe HTMX onLoad registration with error handling
if (typeof htmx !== 'undefined' && htmx.onLoad) {
    htmx.onLoad(initializeSortables);
    console.log('HTMX onLoad handler registered successfully');
} else {
    console.error('HTMX not available or onLoad method missing');
    // Fallback initialization
    document.addEventListener('DOMContentLoaded', function() {
        initializeSortables(document.body);
    });
}

// Global function to manually reinitialize drag and drop for debugging
window.reinitializeDragAndDrop = function() {
    console.log('Manual drag and drop reinitialization requested');
    // Use our safe initialization function
    initializeSortables(document.body);
};

// Global function to re-enable all sortables
window.enableAllSortables = function() {
    console.log('Re-enabling all sortable instances...');
    let enabledCount = 0;
    
    document.querySelectorAll('[data-sortable="tasks"]').forEach(container => {
        if (container.sortableInstance && typeof container.sortableInstance.option === 'function') {
            container.sortableInstance.option("disabled", false);
            enabledCount++;
        }
    });
    
    console.log(`Re-enabled ${enabledCount} sortable instances`);
    return enabledCount;
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
                    input.selectedIndex = 0; // Reset to first option (empty)
                }
            });
        }
    }
}

// Inline task form validation handler
function handleInlineTaskFormSubmit(event) {
    const form = event.target;

    // Get all required field values (tags is optional)
    const title = form.querySelector('input[name="title"]')?.value.trim();
    const priority = form.querySelector('select[name="priority"]')?.value;
    const deadline = form.querySelector('input[name="deadline"]')?.value;

    // Check if at least one assignee is selected
    const assigneeCheckboxes = form.querySelectorAll('.inline-assignee-checkbox:checked');
    const hasAssignee = assigneeCheckboxes.length > 0;

    const missingFields = [];
    if (!title) missingFields.push('Task Title');
    if (!priority) missingFields.push('Priority');
    if (!hasAssignee) missingFields.push('At least one Assignee');
    if (!deadline) missingFields.push('Deadline');

    if (missingFields.length > 0) {
        event.preventDefault();
        event.detail.shouldSubmit = false;

        // Reset button state immediately to prevent stuck "Loading..." state
        const submitButton = form.querySelector('button[type="submit"]');
        if (submitButton) {
            submitButton.disabled = false;
            submitButton.classList.remove('htmx-request');
            // Reset button text if it was changed by HTMX
            const originalText = submitButton.getAttribute('data-original-text') || 'Add Task';
            submitButton.textContent = originalText;
        }

        // Show error notification
        if (typeof showNotification === 'function') {
            showNotification(`Please fill in the following required fields: ${missingFields.join(', ')}`, 'error');
        } else {
            alert(`Please fill in the following required fields:\n\n${missingFields.map(f => '• ' + f).join('\n')}`);
        }

        // Highlight missing fields with red border
        if (!title) {
            const titleInput = form.querySelector('input[name="title"]');
            if (titleInput) titleInput.classList.add('border-red-500', 'border-2');
        }
        if (!priority) {
            const prioritySelect = form.querySelector('select[name="priority"]');
            if (prioritySelect) prioritySelect.classList.add('border-red-500', 'border-2');
        }
        if (!deadline) {
            const deadlineInput = form.querySelector('input[name="deadline"]');
            if (deadlineInput) deadlineInput.classList.add('border-red-500', 'border-2');
        }
        if (!hasAssignee) {
            const assigneeContainer = form.querySelector('.inline-assignee-checkbox')?.closest('.border');
            if (assigneeContainer) assigneeContainer.classList.add('border-red-500', 'border-2');
        }

        // Remove red borders when user starts filling the fields
        setTimeout(() => {
            form.querySelectorAll('.border-red-500').forEach(el => {
                const eventType = el.tagName.toLowerCase() === 'div' ? 'click' : 'input';
                el.addEventListener(eventType, function() {
                    this.classList.remove('border-red-500', 'border-2');
                }, { once: true });
                el.addEventListener('change', function() {
                    this.classList.remove('border-red-500', 'border-2');
                }, { once: true });
            });
        }, 100);

        return false;
    }
}

// Inline task form error handler
function handleInlineTaskFormError(event) {
    console.log('Inline task form error:', event.detail);

    // Reset button state if submission failed
    const form = event.target?.closest('form');
    if (form) {
        const submitButton = form.querySelector('button[type="submit"]');
        if (submitButton) {
            submitButton.disabled = false;
            submitButton.classList.remove('htmx-request');
            const originalText = submitButton.getAttribute('data-original-text') || 'Add Task';
            submitButton.textContent = originalText;
        }
    }

    // Show error notification
    if (typeof showNotification === 'function') {
        showNotification('Failed to create task. Please check all required fields and try again.', 'error');
    }
}

// =============================================================================
// TASK MODAL FUNCTIONS - Complete Implementation
// =============================================================================
// 
// This section provides a comprehensive task editing modal system with:
// - Real-time form validation with visual feedback
// - Auto-save draft functionality to prevent data loss
// - Enhanced keyboard shortcuts (Ctrl+Enter to save, Ctrl+D to delete)
// - Loading states and error handling
// - Accessibility features and user feedback
// 
// Key Functions:
// - showTaskModal(taskId): Entry point for opening task edit modal
// - openTaskDetails(taskId): Loads task data and displays modal
// - saveTaskChanges(taskId): Saves changes with validation
// - validateTaskModal(): Real-time form validation
// - saveDraftForTask/loadDraftForTask: Auto-save functionality
// =============================================================================

function showTaskModal(taskId) {
    console.log('Opening task modal for:', taskId);
    // Delegate to the fully implemented openTaskDetails function
    openTaskDetails(taskId);
}

// =============================================================================
// NESTED BOARD FUNCTIONS - Complete Implementation
// =============================================================================
// 
// This section handles the creation and management of nested boards.
// Nested boards are sub-boards created from existing tasks, providing
// hierarchical project organization with full board functionality.
// =============================================================================

function createNestedBoard(taskId) {
    console.log('Creating nested board for task:', taskId);
    
    if (!taskId) {
        console.error('Task ID is required to create nested board');
        showNotification('Task ID is required to create nested board', 'error');
        return;
    }
    
    const confirmMessage = 'Create a nested board from this task?\n\nThis will:\n• Create a new board using the task title and description\n• Set the current board as the parent board\n• Redirect you to the new board\n• Preserve all task information';
    
    if (!confirm(confirmMessage)) {
        return;
    }
    
    // Show loading state
    showNotification('Creating nested board...', 'info');
    
    console.log(`Creating nested board from task: ${taskId}`);
    
    fetch(`/tasks/${taskId}/convert-to-board`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/x-www-form-urlencoded',
        },
        credentials: 'include'
    }).then(response => {
        console.log('Nested board creation response:', response.status);
        
        if (response.ok) {
            // Check for redirect header (HX-Redirect)
            const redirect = response.headers.get('HX-Redirect');
            console.log('Redirect URL:', redirect);
            
            if (redirect) {
                showNotification('Nested board created successfully! Redirecting...', 'success');
                // Small delay to let user see the success message
                setTimeout(() => {
                    window.location.href = redirect;
                }, 1000);
            } else {
                // Fallback: reload current page
                showNotification('Nested board created successfully!', 'success');
                setTimeout(() => {
                    location.reload();
                }, 1000);
            }
        } else {
            throw new Error(`Server responded with status ${response.status}`);
        }
    }).catch(error => {
        console.error('Error creating nested board:', error);
        showNotification('Failed to create nested board. Please try again.', 'error');
    });
}

// Function to navigate back to parent board (if current board is nested)
function navigateToParentBoard(parentBoardId) {
    if (!parentBoardId) {
        console.warn('No parent board ID provided');
        return;
    }
    
    const confirmMessage = 'Navigate back to parent board?';
    if (confirm(confirmMessage)) {
        window.location.href = `/boards/${parentBoardId}`;
    }
}

// Function to list all nested boards for a given board
function listNestedBoards(boardId) {
    if (!boardId) {
        console.error('Board ID is required to list nested boards');
        return;
    }
    
    console.log(`Fetching nested boards for board: ${boardId}`);
    
    return fetch(`/api/boards/${boardId}/nested`, {
        method: 'GET',
        headers: {
            'Content-Type': 'application/json',
        },
        credentials: 'include'
    }).then(response => {
        if (response.ok) {
            return response.json();
        } else {
            throw new Error(`Failed to fetch nested boards: ${response.status}`);
        }
    }).then(nestedBoards => {
        console.log(`Found ${nestedBoards.length} nested boards`);
        return nestedBoards;
    }).catch(error => {
        console.error('Error fetching nested boards:', error);
        return [];
    });
}

// Enhanced modal functions for creating nested boards from tasks
function showCreateNestedBoardModal() {
    const modal = document.getElementById('create-nested-board-modal');
    if (modal) {
        modal.classList.remove('hidden');
        // Focus on the task select dropdown
        const taskSelect = modal.querySelector('#parent-task-select');
        if (taskSelect) {
            setTimeout(() => taskSelect.focus(), 100);
        }
    }
}

function closeCreateNestedBoardModal() {
    const modal = document.getElementById('create-nested-board-modal');
    if (modal) {
        modal.classList.add('hidden');
        // Reset the form
        const taskSelect = document.getElementById('parent-task-select');
        const taskPreview = document.getElementById('task-preview');
        const createButton = document.getElementById('create-suboard-btn');
        
        if (taskSelect) taskSelect.value = '';
        if (taskPreview) taskPreview.classList.add('hidden');
        if (createButton) {
            createButton.disabled = true;
            createButton.textContent = 'Create Sub-board';
        }
    }
}

// Load tasks and show modal with populated dropdown
function loadTasksAndShowModal() {
    const boardId = getCurrentBoardId();
    if (!boardId) {
        console.error('Board ID not found');
        showNotification('Unable to load tasks. Please refresh the page.', 'error');
        return;
    }
    
    showNotification('Loading tasks...', 'info');
    
    fetch(`/api/boards/${boardId}/tasks`, {
        method: 'GET',
        headers: {
            'Content-Type': 'application/json',
        },
        credentials: 'include'
    }).then(response => {
        if (response.ok) {
            return response.json();
        } else {
            throw new Error(`Failed to fetch tasks: ${response.status}`);
        }
    }).then(tasks => {
        console.log(`Loaded ${tasks.length} tasks for board ${boardId}`);
        populateTaskDropdown(tasks);
        showCreateNestedBoardModal();
    }).catch(error => {
        console.error('Error loading tasks:', error);
        showNotification('Failed to load tasks. Please try again.', 'error');
    });
}

// Populate the task dropdown with available tasks
function populateTaskDropdown(tasks) {
    const taskSelect = document.getElementById('parent-task-select');
    if (!taskSelect) return;
    
    // Clear existing options except the first one
    taskSelect.innerHTML = '<option value="">Select a task to convert to sub-board...</option>';
    
    if (tasks.length === 0) {
        const option = document.createElement('option');
        option.value = '';
        option.textContent = 'No tasks available';
        option.disabled = true;
        taskSelect.appendChild(option);
        return;
    }
    
    // Add tasks to dropdown
    tasks.forEach(task => {
        const option = document.createElement('option');
        option.value = task.id;
        option.textContent = `${task.title} (${getPriorityDisplayName(task.priority)})`;
        option.dataset.title = task.title;
        option.dataset.description = task.description || '';
        option.dataset.priority = task.priority;
        taskSelect.appendChild(option);
    });
    
    // Add event listener for task selection
    taskSelect.addEventListener('change', handleTaskSelection);
}

// Handle task selection and show preview
function handleTaskSelection(event) {
    const selectedOption = event.target.selectedOptions[0];
    const taskPreview = document.getElementById('task-preview');
    const previewContent = document.getElementById('preview-content');
    const createButton = document.getElementById('create-suboard-btn');
    
    if (!selectedOption || !selectedOption.value) {
        taskPreview.classList.add('hidden');
        createButton.disabled = true;
        return;
    }
    
    const title = selectedOption.dataset.title;
    const description = selectedOption.dataset.description;
    const priority = selectedOption.dataset.priority;
    
    // Show preview
    previewContent.innerHTML = `
        <div class="space-y-2">
            <div>
                <span class="font-medium">Title:</span> ${title}
            </div>
            ${description ? `<div><span class="font-medium">Description:</span> ${description}</div>` : ''}
            <div>
                <span class="font-medium">Priority:</span> 
                <span class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${getPriorityBadgeClass(priority)}">
                    ${priority}
                </span>
            </div>
        </div>
    `;
    
    taskPreview.classList.remove('hidden');
    createButton.disabled = false;
}

// Create sub-board from selected task
function createSubBoardFromSelectedTask() {
    const taskSelect = document.getElementById('parent-task-select');
    const createButton = document.getElementById('create-suboard-btn');
    
    if (!taskSelect || !taskSelect.value) {
        showNotification('Please select a task first.', 'error');
        return;
    }
    
    const taskId = taskSelect.value;
    
    // Show loading state
    if (createButton) {
        createButton.disabled = true;
        createButton.textContent = 'Creating...';
    }
    
    // Close modal and create sub-board
    closeCreateNestedBoardModal();
    createNestedBoard(taskId);
}

// Get current board ID from URL or data attributes
function getCurrentBoardId() {
    console.log('Attempting to get board ID...');
    
    // Try to get from URL path first (most reliable)
    const pathMatch = window.location.pathname.match(/\/boards\/([a-f0-9\-]{36})/);
    if (pathMatch) {
        console.log('Board ID found in URL:', pathMatch[1]);
        return pathMatch[1];
    }
    
    // Try to get from main board container
    const boardContainer = document.getElementById('board-container');
    if (boardContainer && boardContainer.dataset.boardId) {
        console.log('Board ID found in board container:', boardContainer.dataset.boardId);
        return boardContainer.dataset.boardId;
    }
    
    // Try to get from any element with data-board-id
    const boardElement = document.querySelector('[data-board-id]');
    if (boardElement && boardElement.dataset.boardId) {
        console.log('Board ID found in DOM element:', boardElement.dataset.boardId);
        return boardElement.dataset.boardId;
    }
    
    // Try to get from add column modal
    const addColumnModal = document.getElementById('add-column-modal');
    if (addColumnModal) {
        const hiddenInput = addColumnModal.querySelector('input[name="board_id"]');
        if (hiddenInput && hiddenInput.value) {
            console.log('Board ID found in add column modal:', hiddenInput.value);
            return hiddenInput.value;
        }
    }
    
    console.error('Could not determine current board ID');
    console.log('Current URL:', window.location.pathname);
    console.log('Available elements with data-board-id:', document.querySelectorAll('[data-board-id]'));
    return null;
}

// Helper function to get priority display name
function getPriorityDisplayName(priority) {
    const names = {
        'Urgent': 'Urgent',
        'High': 'High',
        'Medium': 'Medium',
        'Low': 'Low'
    };
    return names[priority] || priority;
}

// Function to check for tasks and show/hide create sub-board button
function checkTasksAndUpdateButton() {
    const boardId = getCurrentBoardId();
    if (!boardId) {
        console.log('Skipping task check - not on a board page');
        return;
    }
    
    fetch(`/api/boards/${boardId}/tasks`, {
        method: 'GET',
        headers: {
            'Content-Type': 'application/json',
        },
        credentials: 'include'
    }).then(response => {
        if (response.ok) {
            return response.json();
        }
        return [];
    }).then(tasks => {
        console.log('Tasks API response:', tasks, 'Type:', typeof tasks);
        const createButton = document.getElementById('create-subboard-button');
        if (createButton) {
            if (tasks && Array.isArray(tasks) && tasks.length > 0) {
                createButton.classList.remove('hidden');
                createButton.classList.add('inline-flex');
            } else {
                createButton.classList.add('hidden');
                createButton.classList.remove('inline-flex');
            }
        }
    }).catch(error => {
        console.log('Could not check tasks:', error);
        // Hide button on error
        const createButton = document.getElementById('create-subboard-button');
        if (createButton) {
            createButton.classList.add('hidden');
            createButton.classList.remove('inline-flex');
        }
    });
}

// Update task count in column headers
function updateTaskCount(columnId) {
    const column = document.querySelector(`[data-column-id="${columnId}"]`);
    if (column) {
        const tasks = column.querySelectorAll('.task-card');
        let countElement = column.querySelector('.bg-theme-secondary.text-theme-primary');
        
        // Fallback: try finding by more general selector if specific one doesn't work
        if (!countElement) {
            countElement = column.querySelector('span.px-2.py-1.rounded-full');
        }
        
        if (countElement) {
            console.log(`Updating task count for column ${columnId}: ${tasks.length}`);
            countElement.textContent = tasks.length;
        } else {
            console.warn(`Could not find task count element for column ${columnId}`);
            console.log('Available elements in column:', column.innerHTML.substring(0, 200) + '...');
        }
    } else {
        console.warn(`Could not find column with id ${columnId}`);
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

// Enhanced keyboard shortcuts
document.addEventListener('keydown', function(e) {
    // Don't trigger shortcuts when typing in inputs or textareas
    if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
        return;
    }

    // Ctrl/Cmd + K is reserved for global search - DO NOT use for task creation
    // The global search handler is defined in global_header.templ and search_modal.templ
    
    // Escape to close modals/forms
    if (e.key === 'Escape') {
        closeAllModals();
        closeCreateNestedBoardModal();
    }
    
    // Ctrl/Cmd + Enter to save task changes in modal
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
        const taskModal = document.getElementById('task-modal');
        if (taskModal && !taskModal.classList.contains('hidden')) {
            e.preventDefault();
            const taskId = taskModal.dataset.taskId;
            if (taskId) {
                saveTaskChanges(taskId);
            }
        }
    }
    
    // Ctrl/Cmd + D to delete task in modal
    if ((e.ctrlKey || e.metaKey) && e.key === 'd' && !e.shiftKey) {
        const taskModal = document.getElementById('task-modal');
        if (taskModal && !taskModal.classList.contains('hidden')) {
            e.preventDefault();
            const taskId = taskModal.dataset.taskId;
            if (taskId) {
                deleteTask(taskId);
            }
        }
    }

    // Ctrl/Cmd + Shift + D to toggle dark mode
    if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'D') {
        e.preventDefault();
        toggleDarkMode();
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
    const url = evt.detail.pathInfo?.path || evt.detail.xhr?.responseURL || '';
    
    if (evt.detail.xhr.status === 200) {
        console.log('HTMX request successful for URL:', url);
        
        // Only update task counts for task-related operations
        if (url.includes('/tasks') || url.includes('/columns') || url.includes('/boards')) {
            console.log('Updating task counts for all columns after task/column/board operation');
            // Update task counts and empty states for all columns
            document.querySelectorAll('[data-column-id]').forEach(column => {
                const columnId = column.dataset.columnId;
                updateTaskCount(columnId);
                updateEmptyState(columnId);
            });
        }
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
            // Update active tasks count before reload
            if (isCompleted) {
                // Reopening a completed task - increment active tasks
                updateActiveTasksCount(1);
            } else {
                // Completing a task - decrement active tasks
                updateActiveTasksCount(-1);
            }

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
        
        // Setup real-time form validation
        setupTaskModalValidation();
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
                'HX-Request': 'true'
            },
            credentials: 'include'
        }).then(response => {
            if (response.ok) {
                console.log('Task deleted successfully');
                
                // Check for HX-Trigger header to handle nested board deletion
                const hxTrigger = response.headers.get('HX-Trigger');
                let deletedNestedBoardId = null;
                
                if (hxTrigger) {
                    console.log('HX-Trigger header found:', hxTrigger);
                    const triggers = hxTrigger.split(',').map(t => t.trim());
                    console.log('Parsed triggers:', triggers);
                    
                    triggers.forEach(trigger => {
                        if (trigger.startsWith('nestedBoardDeleted-')) {
                            deletedNestedBoardId = trigger.replace('nestedBoardDeleted-', '');
                            console.log('Nested board deletion detected for board:', deletedNestedBoardId);
                        }
                    });
                }
                
                // Remove the task card from DOM with animation
                if (taskCard) {
                    // Check if the task was active (not completed) before deletion
                    const wasActiveTask = taskCard.querySelector('.text-green-500') === null;

                    // Check if the task was due soon before deletion
                    const deadline = getTaskDeadline(taskCard);
                    const wasDueSoon = deadline && (deadline === 'soon' || isTaskDueSoon(deadline));

                    taskCard.style.transition = 'all 0.3s ease';
                    taskCard.style.opacity = '0';
                    taskCard.style.transform = 'scale(0.8)';

                    setTimeout(() => {
                        taskCard.remove();

                        // Update active tasks count if it was an active task
                        if (wasActiveTask) {
                            updateActiveTasksCount(-1);
                        }

                        // Update due soon count if it was due soon
                        if (wasDueSoon) {
                            updateDueSoonCount(-1);
                        }

                        // Update task counts, empty states, and heights immediately
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
                        
                        // Trigger height equalization immediately after task deletion for real-time shrinking
                        console.log('Triggering immediate height equalization after task deletion');
                        equalizeColumnHeightsOptimized();
                    }, 100); // Reduced from 300ms to 100ms for faster response
                }
                
                // Handle nested board removal if detected
                if (deletedNestedBoardId) {
                    console.log('Also processing nested board deletion for:', deletedNestedBoardId);
                    
                    // Debug: List all elements with data-board-id attributes
                    const allBoardElements = document.querySelectorAll('[data-board-id]');
                    console.log('All elements with data-board-id:', allBoardElements);
                    allBoardElements.forEach((el, index) => {
                        console.log(`Element ${index}:`, el, 'data-board-id:', el.getAttribute('data-board-id'));
                    });
                    
                    // Find and remove the nested board card/tile
                    const nestedBoardCard = document.querySelector(`[data-board-id="${deletedNestedBoardId}"]`);
                    console.log('Looking for nested board card with selector:', `[data-board-id="${deletedNestedBoardId}"]`);
                    console.log('Found nested board card:', nestedBoardCard);
                    
                    if (nestedBoardCard) {
                        console.log('Removing nested board card from UI:', deletedNestedBoardId);
                        nestedBoardCard.style.transition = 'all 0.3s ease';
                        nestedBoardCard.style.opacity = '0';
                        nestedBoardCard.style.transform = 'scale(0.8)';
                        setTimeout(() => {
                            nestedBoardCard.remove();
                            console.log('Nested board card removed from DOM');
                        }, 300);
                    } else {
                        console.log('No nested board card found in DOM for ID:', deletedNestedBoardId);
                        console.log('Available board cards:', document.querySelectorAll('[data-board-id]'));
                    }
                    
                    showNotification('Task and its nested board deleted successfully!', 'success');
                } else {
                    showNotification('Task deleted successfully!', 'success');
                }
                
                // Update sub-board button visibility since task count changed
                setTimeout(checkTasksAndUpdateButton, 300);
                
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
        
        // Show loading notification
        showNotification('Deleting board...', 'info');
        
        const requestUrl = `/boards/${boardId}`;
        console.log('Full request URL:', window.location.origin + requestUrl);
        
        fetch(requestUrl, {
            method: 'DELETE',
            headers: {
                'Content-Type': 'application/json',
                'HX-Request': 'true'
            },
            credentials: 'include'
        }).then(response => {
            console.log('Response received!');
            console.log('Response status:', response.status);
            console.log('Response ok:', response.ok);
            console.log('Response headers:', [...response.headers.entries()]);
            
            if (response.ok) {
                console.log('Board deletion successful');
                showNotification('Board deleted successfully!', 'success');
                
                // Remove the board card immediately for better UX (works on dashboard and main board)
                const boardCard = document.querySelector(`[data-board-id="${boardId}"]`);
                if (boardCard) {
                    // Determine if this is in main boards or nested boards grid
                    const isInMainGrid = boardCard.closest('#boards-grid') !== null;
                    const isInNestedGrid = boardCard.closest('#nested-boards-grid') !== null;

                    boardCard.style.transition = 'all 0.3s ease';
                    boardCard.style.opacity = '0';
                    boardCard.style.transform = 'scale(0.8)';
                    setTimeout(() => {
                        boardCard.remove();

                        // Check if there are any remaining board cards in the main boards grid
                        const boardsGrid = document.getElementById('boards-grid');
                        if (boardsGrid) {
                            const remainingMainCards = boardsGrid.querySelectorAll('[data-board-id]');
                            if (remainingMainCards.length === 0) {
                                // Show empty state
                                const emptyState = document.createElement('div');
                                emptyState.id = 'empty-state';
                                emptyState.className = 'col-span-full text-center py-12';
                                emptyState.innerHTML = `
                                    <svg class="mx-auto h-12 w-12 text-theme-muted transition-colors duration-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2zM8 7v10M16 7v10"></path>
                                    </svg>
                                    <h3 class="mt-2 text-sm font-medium text-theme-primary transition-colors duration-300">No boards</h3>
                                    <p class="mt-1 text-sm text-theme-secondary transition-colors duration-300">Get started by creating your first board.</p>
                                    <div class="mt-6">
                                        <button
                                            onclick="document.getElementById('create-board-modal').classList.remove('hidden')"
                                            class="inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-terracotta-600 dark:bg-yinmn-blue-600 hover:bg-terracotta-700 dark:hover:bg-yinmn-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-terracotta-500 dark:focus:ring-yinmn-blue-500 transition-all duration-300"
                                        >
                                            <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
                                            </svg>
                                            Create Board
                                        </button>
                                    </div>
                                `;
                                boardsGrid.appendChild(emptyState);
                            }

                            // Update main boards count if deleted from main grid
                            if (isInMainGrid) {
                                updateMainBoardsCount(-1);
                            }
                        }

                        // Check if there are any remaining nested board cards
                        const nestedBoardsGrid = document.getElementById('nested-boards-grid');
                        if (nestedBoardsGrid) {
                            const remainingNestedCards = nestedBoardsGrid.querySelectorAll('[data-board-id]');
                            if (remainingNestedCards.length === 0) {
                                // Hide/remove the entire nested boards section
                                const nestedBoardsSection = document.getElementById('nested-boards-section');
                                if (nestedBoardsSection) {
                                    nestedBoardsSection.style.transition = 'all 0.3s ease';
                                    nestedBoardsSection.style.opacity = '0';
                                    nestedBoardsSection.style.transform = 'scale(0.98)';
                                    setTimeout(() => {
                                        nestedBoardsSection.remove();
                                    }, 300);
                                }
                            }

                            // Update nested boards count if deleted from nested grid
                            if (isInNestedGrid) {
                                updateNestedBoardsCount(-1);
                            }
                        }

                        // Update the total boards stat
                        updateTotalBoardsCount(-1);
                    }, 300);
                }
                
                // Check for HX-Redirect header (only present if we should redirect)
                const redirect = response.headers.get('HX-Redirect');
                if (redirect) {
                    console.log('HX-Redirect found, redirecting to:', redirect);
                    // Redirect after animation if server says we should
                    setTimeout(() => {
                        window.location.href = redirect;
                    }, 500);
                } else {
                    console.log('No redirect requested, staying on current page');
                    // No redirect - just remove the board card and stay on current page
                }
            } else {
                return response.text().then(text => {
                    console.error('Deletion failed with status:', response.status);
                    console.error('Deletion failed with response:', text);
                    showNotification(`Failed to delete board. Status: ${response.status}. Please try again.`, 'error');
                });
            }
        }).catch(error => {
            console.error('Network error details:');
            console.error('Error name:', error.name);
            console.error('Error message:', error.message);
            console.error('Error stack:', error.stack);
            console.error('Full error object:', error);
            showNotification('Failed to delete board. Network error. Please try again.', 'error');
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

// Enhanced form validation for task modal
function validateTaskModal() {
    const title = document.getElementById('task-title');
    const description = document.getElementById('task-description');
    const priority = document.getElementById('task-priority');
    const deadline = document.getElementById('task-deadline');
    
    let isValid = true;
    const errors = [];

    // Clear previous validation styles
    [title, description, priority, deadline].forEach(field => {
        if (field) {
            field.classList.remove('border-red-500', 'border-green-500');
        }
    });

    // Validate title
    if (!title?.value?.trim()) {
        title?.classList.add('border-red-500');
        errors.push('Task title is required');
        isValid = false;
    } else if (title.value.trim().length > 200) {
        title.classList.add('border-red-500');
        errors.push('Task title must be less than 200 characters');
        isValid = false;
    } else {
        title.classList.add('border-green-500');
    }

    // Validate description length
    if (description?.value && description.value.length > 1000) {
        description.classList.add('border-red-500');
        errors.push('Description must be less than 1000 characters');
        isValid = false;
    } else if (description?.value) {
        description.classList.add('border-green-500');
    }

    // Validate deadline (if provided)
    if (deadline?.value) {
        const deadlineDate = new Date(deadline.value);
        const now = new Date();
        const completed = document.getElementById('task-completed')?.checked || false;

        // Only validate past deadlines for non-completed tasks
        // Completed tasks can have past deadlines (they were likely completed late)
        if (!completed && deadlineDate < now && Math.abs(deadlineDate - now) > 60000) { // Allow 1 minute buffer for current time
            deadline.classList.add('border-red-500');
            errors.push('Deadline cannot be in the past');
            isValid = false;
        } else {
            deadline.classList.add('border-green-500');
        }
    }

    return { isValid, errors };
}

// Real-time validation setup for task modal
function setupTaskModalValidation() {
    const title = document.getElementById('task-title');
    const description = document.getElementById('task-description');
    const deadline = document.getElementById('task-deadline');
    const priority = document.getElementById('task-priority');
    const assignee = document.getElementById('task-assignee');
    const completed = document.getElementById('task-completed');

    // Add real-time validation
    [title, description, deadline].forEach(field => {
        if (field) {
            field.addEventListener('blur', validateTaskModal);
            field.addEventListener('input', debounce(validateTaskModal, 300));
        }
    });

    // Add auto-save draft functionality
    const taskModal = document.getElementById('task-modal');
    const taskId = taskModal?.dataset?.taskId;
    
    if (taskId) {
        const allFields = [title, description, deadline, priority, assignee, completed];
        allFields.forEach(field => {
            if (field) {
                field.addEventListener('input', debounce(() => {
                    saveDraftForTask(taskId);
                }, 1000));
                field.addEventListener('change', debounce(() => {
                    saveDraftForTask(taskId);
                }, 1000));
            }
        });

        // Load existing draft if available
        loadDraftForTask(taskId);
    }
}

// Save draft for task modal
function saveDraftForTask(taskId) {
    if (!taskId) return;
    
    const draftData = {
        title: document.getElementById('task-title')?.value || '',
        description: document.getElementById('task-description')?.value || '',
        priority: document.getElementById('task-priority')?.value || '',
        deadline: document.getElementById('task-deadline')?.value || '',
        assigneeId: document.getElementById('task-assignee')?.value || '',
        completed: document.getElementById('task-completed')?.checked || false,
        timestamp: new Date().toISOString()
    };
    
    saveDraft(`task_${taskId}`, draftData);
}

// Load draft for task modal
function loadDraftForTask(taskId) {
    if (!taskId) return;
    
    const draft = loadDraft(`task_${taskId}`);
    if (!draft) return;
    
    // Only load draft if it's recent (within last hour)
    const draftTime = new Date(draft.timestamp);
    const now = new Date();
    const hoursDiff = (now - draftTime) / (1000 * 60 * 60);
    
    if (hoursDiff > 1) {
        clearDraft(`task_${taskId}`);
        return;
    }
    
    // Show draft restoration notification
    const restoreDraft = confirm('Found unsaved changes from your previous edit session. Would you like to restore them?');
    
    if (restoreDraft) {
        const title = document.getElementById('task-title');
        const description = document.getElementById('task-description');
        const priority = document.getElementById('task-priority');
        const deadline = document.getElementById('task-deadline');
        const assignee = document.getElementById('task-assignee');
        const completed = document.getElementById('task-completed');
        
        if (title && draft.title !== title.value) title.value = draft.title;
        if (description && draft.description !== description.value) description.value = draft.description;
        if (priority && draft.priority !== priority.value) priority.value = draft.priority;
        if (deadline && draft.deadline !== deadline.value) deadline.value = draft.deadline;
        if (assignee && draft.assigneeId !== assignee.value) assignee.value = draft.assigneeId;
        if (completed && draft.completed !== completed.checked) completed.checked = draft.completed;
        
        showNotification('Draft restored successfully!', 'info');
    } else {
        clearDraft(`task_${taskId}`);
    }
}

// Save task changes from modal with enhanced validation
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

    // Validate form before submission
    const validation = validateTaskModal();
    if (!validation.isValid) {
        // Show validation errors
        const errorMessage = validation.errors.join('\n');
        showNotification(errorMessage, 'error');
        
        // Focus on first invalid field
        const firstInvalidField = document.querySelector('.border-red-500');
        if (firstInvalidField) {
            firstInvalidField.focus();
        }
        return;
    }

    // Show loading state on save button
    const saveButton = form.querySelector('button[onclick*="saveTaskChanges"], button[data-action="save-changes"]');
    const originalText = saveButton ? saveButton.textContent : 'Save Changes';
    if (saveButton) {
        saveButton.disabled = true;
        saveButton.textContent = 'Saving...';
        saveButton.classList.add('opacity-50', 'cursor-not-allowed');
    }

    // Gather form data
    const title = document.getElementById('task-title')?.value?.trim() || '';
    const description = document.getElementById('task-description')?.value?.trim() || '';
    const priority = document.getElementById('task-priority')?.value || '';
    const deadline = document.getElementById('task-deadline')?.value || '';
    const assigneeId = document.getElementById('task-assignee')?.value || '';
    const completed = document.getElementById('task-completed')?.checked || false;

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
            
            // Clear form validation styles
            document.querySelectorAll('.border-red-500, .border-green-500').forEach(field => {
                field.classList.remove('border-red-500', 'border-green-500');
            });
            
            // Clear saved draft since changes were successfully saved
            clearDraft(`task_${taskId}`);
            
            // Update sub-board button visibility in case task count changed
            setTimeout(checkTasksAndUpdateButton, 200);
            
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
            saveButton.classList.remove('opacity-50', 'cursor-not-allowed');
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

// Nested Board Menu Functions (for main board view)
function toggleNestedBoardMenu(boardId) {
    // Hide all other nested board menus first
    const allMenus = document.querySelectorAll('[id^="nested-board-menu-"]');
    allMenus.forEach(menu => {
        if (menu.id !== `nested-board-menu-${boardId}`) {
            menu.classList.add('hidden');
        }
    });
    
    // Toggle the specific menu
    const menu = document.getElementById(`nested-board-menu-${boardId}`);
    if (menu) {
        menu.classList.toggle('hidden');
    }
}

// Close nested board menus when clicking outside
document.addEventListener('click', function(event) {
    if (!event.target.closest('[id^="nested-board-menu-"]') && !event.target.closest('button[onclick*="toggleNestedBoardMenuScript"]')) {
        document.querySelectorAll('[id^="nested-board-menu-"]').forEach(menu => {
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
window.toggleNestedBoardMenu = toggleNestedBoardMenu;
window.deleteTask = deleteTask;
window.deleteColumn = deleteColumn;
window.deleteBoard = deleteBoard;
window.closeTaskModal = closeTaskModal;
window.saveTaskChanges = saveTaskChanges;
window.convertToSubBoard = convertToSubBoard;
window.copyTaskLink = copyTaskLink;
window.toggleDarkMode = toggleDarkMode;

// Export enhanced task modal functions
window.validateTaskModal = validateTaskModal;
window.setupTaskModalValidation = setupTaskModalValidation;
window.saveDraftForTask = saveDraftForTask;
window.loadDraftForTask = loadDraftForTask;

// Export nested board functions
window.createNestedBoard = createNestedBoard;
window.navigateToParentBoard = navigateToParentBoard;
window.listNestedBoards = listNestedBoards;
window.showCreateNestedBoardModal = showCreateNestedBoardModal;
window.closeCreateNestedBoardModal = closeCreateNestedBoardModal;
window.loadTasksAndShowModal = loadTasksAndShowModal;
window.createSubBoardFromSelectedTask = createSubBoardFromSelectedTask;
window.checkTasksAndUpdateButton = checkTasksAndUpdateButton;

// Export debug functions
window.debugSortables = debugSortables;
window.reinitializeDragAndDrop = reinitializeDragAndDrop;
window.enableAllSortables = enableAllSortables;

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

// Debug: Log all HTMX events to understand what's happening
document.body.addEventListener('htmx:afterRequest', function(evt) {
    console.log('HTMX after request:', {
        requestConfig: evt.detail.requestConfig,
        xhr: evt.detail.xhr,
        target: evt.detail.target,
        responseHeaders: [...evt.detail.xhr.getAllResponseHeaders().split('\r\n')].filter(h => h.includes('HX-Trigger'))
    });
    
    // Check for HX-Trigger header manually
    const hxTrigger = evt.detail.xhr.getResponseHeader('HX-Trigger');
    if (hxTrigger) {
        console.log('HX-Trigger header found:', hxTrigger);
        
        // Parse the triggers
        const triggers = hxTrigger.split(',').map(t => t.trim());
        console.log('Parsed triggers:', triggers);
        
        let taskDeleted = false;
        let deletedNestedBoardId = null;
        
        triggers.forEach(trigger => {
            if (trigger === 'taskDeleted') {
                taskDeleted = true;
                console.log('Task deletion detected');
            } else if (trigger.startsWith('nestedBoardDeleted-')) {
                deletedNestedBoardId = trigger.replace('nestedBoardDeleted-', '');
                console.log('Nested board deletion detected for board:', deletedNestedBoardId);
            }
        });
        
        if (taskDeleted) {
            console.log('Processing task deletion...');
            
            // Task was deleted - update all column task counts and empty states
            document.querySelectorAll('[data-column-id]').forEach(column => {
                const columnId = column.dataset.columnId;
                updateTaskCount(columnId);
                updateEmptyState(columnId);
            });
            
            // Update sub-board button visibility
            setTimeout(checkTasksAndUpdateButton, 200);
            
            if (deletedNestedBoardId) {
                console.log('Also processing nested board deletion for:', deletedNestedBoardId);
                
                // Debug: List all elements with data-board-id attributes
                const allBoardElements = document.querySelectorAll('[data-board-id]');
                console.log('All elements with data-board-id:', allBoardElements);
                allBoardElements.forEach((el, index) => {
                    console.log(`Element ${index}:`, el, 'data-board-id:', el.getAttribute('data-board-id'));
                });
                
                // Find and remove the nested board card/tile
                const nestedBoardCard = document.querySelector(`[data-board-id="${deletedNestedBoardId}"]`);
                console.log('Looking for nested board card with selector:', `[data-board-id="${deletedNestedBoardId}"]`);
                console.log('Found nested board card:', nestedBoardCard);
                
                if (nestedBoardCard) {
                    console.log('Removing nested board card from UI:', deletedNestedBoardId);
                    nestedBoardCard.style.transition = 'all 0.3s ease';
                    nestedBoardCard.style.opacity = '0';
                    nestedBoardCard.style.transform = 'scale(0.8)';
                    setTimeout(() => {
                        nestedBoardCard.remove();
                        console.log('Nested board card removed from DOM');
                    }, 300);
                } else {
                    console.log('No nested board card found in DOM for ID:', deletedNestedBoardId);
                    console.log('Available board cards:', document.querySelectorAll('[data-board-id]'));
                }
                
                showNotification('Task and its nested board deleted successfully!', 'success');
            } else {
                showNotification('Task deleted successfully!', 'success');
            }
        }
    }
});

// HTMX event listeners for real-time updates (keep the original as backup)
document.body.addEventListener('htmx:trigger', function(evt) {
    console.log('HTMX trigger event received (backup handler):', evt.detail);
});

// Listen for server-sent events when tasks with nested boards are deleted
document.body.addEventListener('htmx:responseError', function(evt) {
    console.error('HTMX Response Error:', evt.detail);
    // Handle errors gracefully with user notifications
    let errorMessage = '';
    
    // Try to get the actual error message from the server response
    if (evt.detail.xhr.responseText && evt.detail.xhr.responseText.trim()) {
        errorMessage = evt.detail.xhr.responseText.trim();
    } else if (evt.detail.xhr.status >= 400 && evt.detail.xhr.status < 500) {
        errorMessage = 'Action failed. Please check your permissions and try again.';
    } else if (evt.detail.xhr.status >= 500) {
        errorMessage = 'Server error occurred. Please try again later.';
    } else {
        errorMessage = 'An unexpected error occurred. Please try again.';
    }
    
    showNotification(errorMessage, 'error');
});

// Column Height Equalization System - Dynamic Height Adjustment
function equalizeColumnHeightsOptimized() {
    const columns = document.querySelectorAll('.kanban-column');
    if (columns.length === 0) return;
    
    console.log('Dynamically adjusting column heights based on current content...');
    
    // Step 1: Reset all columns to their natural content height and measure
    const columnData = [];
    
    columns.forEach((column, index) => {
        const columnContent = column.querySelector('.column-content');
        const tasksContainer = column.querySelector('.tasks-container');
        
        if (columnContent && tasksContainer) {
            // Ensure smooth transitions are enabled on column content (match CSS timing)
            columnContent.style.transition = 'min-height 0.2s ease';
            
            // CRITICAL FIX: Completely reset height constraints to get true natural height
            columnContent.style.minHeight = 'auto';
            columnContent.style.height = 'auto';
            tasksContainer.style.minHeight = 'auto';
            tasksContainer.style.height = 'auto';
            
            columnData.push({
                column,
                columnContent,
                tasksContainer,
                index
            });
        }
    });
    
    // Step 2: Force reflow and measure all natural heights
    let naturalTallestHeight = 0;
    // Get base minimum height from CSS custom property
    const baseMinHeight = parseInt(getComputedStyle(document.documentElement)
        .getPropertyValue('--column-base-min-height')) || 174;
    
    // Force reflow and measure all natural heights immediately  
    columnData.forEach(({ columnContent, index }) => {
        void columnContent.offsetHeight; // Force reflow
        const naturalHeight = columnContent.offsetHeight;
        naturalTallestHeight = Math.max(naturalTallestHeight, naturalHeight);
        
        columnData[index].naturalHeight = naturalHeight;
        console.log(`Column ${index} natural content height: ${naturalHeight}px`);
    });
    
    // The target height is the maximum of base minimum and current tallest natural height
    const targetHeight = Math.max(baseMinHeight, naturalTallestHeight);
    console.log(`Target height for all columns: ${targetHeight}px (natural tallest: ${naturalTallestHeight}px)`);
    
    // Step 3: Apply dynamic heights - both expand AND shrink as needed
    columnData.forEach(({ columnContent, naturalHeight, index }) => {
        // CRITICAL FIX: Always set min-height to target, allowing both expansion and shrinking
        columnContent.style.minHeight = `${targetHeight}px`;
        
        if (naturalHeight < targetHeight) {
            console.log(`Column ${index}: Expanding from ${naturalHeight}px to ${targetHeight}px`);
        } else if (naturalHeight > targetHeight) {
            console.log(`Column ${index}: Shrinking from ${naturalHeight}px to ${targetHeight}px`);
        } else {
            console.log(`Column ${index}: Maintaining height at ${naturalHeight}px`);
        }
    });
    
    console.log(`All ${columnData.length} columns dynamically adjusted to ${targetHeight}px height`);
}

// Debounced version for performance with safeguard
let equalizeHeightsTimeout;
let isEqualizing = false;

function equalizeColumnHeightsDebounced(delay = 100) {
    clearTimeout(equalizeHeightsTimeout);
    
    equalizeHeightsTimeout = setTimeout(() => {
        if (isEqualizing) {
            console.log('Equalization already in progress, skipping...');
            return;
        }
        
        isEqualizing = true;
        try {
            equalizeColumnHeightsOptimized();
        } finally {
            // Quick cooldown for responsive feel
            setTimeout(() => {
                isEqualizing = false;
            }, 150);
        }
    }, delay);
}

// Initialize column height equalization on page load
document.addEventListener('DOMContentLoaded', function() {
    // Initial equalization
    setTimeout(equalizeColumnHeightsDebounced, 300);
    
    // Set up MutationObserver to watch for DOM changes
    const boardColumns = document.getElementById('board-columns');
    if (boardColumns) {
        const observer = new MutationObserver((mutations) => {
            let shouldEqualize = false;
            
            mutations.forEach((mutation) => {
                // Check if tasks were added/removed or columns changed
                if (mutation.type === 'childList') {
                    const addedNodes = Array.from(mutation.addedNodes);
                    const removedNodes = Array.from(mutation.removedNodes);
                    
                    // Check if task cards or columns were added/removed
                    const hasTaskChanges = [...addedNodes, ...removedNodes].some(node => 
                        node.nodeType === Node.ELEMENT_NODE && 
                        (node.matches && (node.matches('.task-card') || node.matches('.kanban-column')))
                    );
                    
                    if (hasTaskChanges) {
                        shouldEqualize = true;
                    }
                }
            });
            
            if (shouldEqualize) {
                equalizeColumnHeightsDebounced(300); // Responsive delay for MutationObserver
            }
        });
        
        observer.observe(boardColumns, {
            childList: true,
            subtree: true
        });
    }
    
    // Also equalize on window resize
    window.addEventListener('resize', () => {
        equalizeColumnHeightsDebounced(300);
    });
    
    // Handle page navigation (including nested board navigation)
    window.addEventListener('popstate', () => {
        setTimeout(() => {
            equalizeColumnHeightsDebounced(300);
        }, 100);
    });
    
    // Handle sub-board loading/switching
    document.addEventListener('click', (e) => {
        // Check if clicked element is a board navigation link
        if (e.target.closest('a[href*="/boards/"]')) {
            // Delay to allow page to load
            setTimeout(() => {
                equalizeColumnHeightsDebounced(500);
            }, 200);
        }
    });
});

// Global function that can be called by other parts of the application
window.equalizeAllColumnHeights = function() {
    if (typeof equalizeColumnHeightsDebounced === 'function') {
        equalizeColumnHeightsDebounced(100);
    }
};

// Simplified force reset function
window.forceResetColumnHeights = function() {
    console.log('Force resetting column heights...');
    equalizeColumnHeightsOptimized();
};

// Handle task creation/deletion events more precisely
document.addEventListener('taskCreated', () => {
    equalizeColumnHeightsDebounced(100);
});

document.addEventListener('taskDeleted', () => {
    equalizeColumnHeightsDebounced(100);
});

document.addEventListener('taskUpdated', () => {
    equalizeColumnHeightsDebounced(100);
});

// Add event listeners for common task operations
document.body.addEventListener('htmx:afterRequest', function(evt) {
    const url = evt.detail.pathInfo?.path || evt.detail.xhr?.responseURL || '';
    console.log('HTMX after request for URL:', url);

    // Check if this was a task-related operation
    if (url.includes('/tasks') || url.includes('/columns')) {
        console.log('Triggering immediate height equalization for task/column operation:', url);
        // Use immediate direct call for real-time column resizing on all task operations
        setTimeout(() => equalizeColumnHeightsOptimized(), 10); // Tiny delay for DOM to settle
    }
});

// Task Assignee Management Functions (for task details modal)
window.handleAssigneeChange = function(checkbox) {
    const taskId = checkbox.dataset.taskId;
    const userId = checkbox.dataset.userId;
    const isChecked = checkbox.checked;

    if (isChecked) {
        // Add assignee
        fetch(`/api/tasks/${taskId}/assignees`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
            },
            body: `user_id=${userId}`,
            credentials: 'include'
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                console.log('Assignee added successfully');

                // Refresh the task card to show the new assignee
                fetch(`/api/tasks/${taskId}/card`, {
                    method: 'GET',
                    credentials: 'include'
                })
                .then(response => response.text())
                .then(html => {
                    const taskCard = document.querySelector(`[data-task-id="${taskId}"]`);
                    if (taskCard) {
                        const tempDiv = document.createElement('div');
                        tempDiv.innerHTML = html;
                        const newTaskCard = tempDiv.firstElementChild;
                        if (newTaskCard) {
                            taskCard.replaceWith(newTaskCard);
                        }
                    }
                })
                .catch(error => {
                    console.error('Error refreshing task card:', error);
                });

                // Reload the modal to show completion checkbox
                setTimeout(() => location.reload(), 500);
            } else {
                checkbox.checked = false;
                alert('Failed to add assignee');
            }
        })
        .catch(error => {
            console.error('Error adding assignee:', error);
            checkbox.checked = false;
            alert('Failed to add assignee');
        });
    } else {
        // Remove assignee
        fetch(`/api/tasks/${taskId}/assignees/${userId}`, {
            method: 'DELETE',
            credentials: 'include'
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                console.log('Assignee removed successfully');

                // Refresh the task card to remove the assignee avatar
                fetch(`/api/tasks/${taskId}/card`, {
                    method: 'GET',
                    credentials: 'include'
                })
                .then(response => response.text())
                .then(html => {
                    const taskCard = document.querySelector(`[data-task-id="${taskId}"]`);
                    if (taskCard) {
                        const tempDiv = document.createElement('div');
                        tempDiv.innerHTML = html;
                        const newTaskCard = tempDiv.firstElementChild;
                        if (newTaskCard) {
                            taskCard.replaceWith(newTaskCard);
                        }
                    }
                })
                .catch(error => {
                    console.error('Error refreshing task card:', error);
                });

                // Reload the modal to hide completion checkbox
                setTimeout(() => location.reload(), 500);
            } else {
                checkbox.checked = true;
                alert('Failed to remove assignee');
            }
        })
        .catch(error => {
            console.error('Error removing assignee:', error);
            checkbox.checked = true;
            alert('Failed to remove assignee');
        });
    }
};

window.handleAssigneeCompletionChange = function(checkbox) {
    const taskId = checkbox.dataset.taskId;
    const completed = checkbox.checked;

    fetch(`/api/tasks/${taskId}/assignee-completion`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: `completed=${completed}`,
        credentials: 'include'
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            console.log('Completion status updated successfully');
            // Show success feedback
            if (typeof showNotification === 'function') {
                showNotification(completed ? 'Marked as complete' : 'Marked as incomplete', 'success');
            }

            // Fetch and replace the task card to show updated completion status
            fetch(`/api/tasks/${taskId}/card`, {
                method: 'GET',
                credentials: 'include'
            })
            .then(response => response.text())
            .then(html => {
                const taskCard = document.querySelector(`[data-task-id="${taskId}"]`);
                if (taskCard) {
                    // Replace the entire task card with the updated HTML
                    const tempDiv = document.createElement('div');
                    tempDiv.innerHTML = html;
                    const newTaskCard = tempDiv.firstElementChild;

                    if (newTaskCard) {
                        taskCard.replaceWith(newTaskCard);
                    }
                }
            })
            .catch(error => {
                console.error('Error refreshing task card:', error);
            });
        } else {
            checkbox.checked = !completed;
            alert('Failed to update completion status');
        }
    })
    .catch(error => {
        console.error('Error updating completion:', error);
        checkbox.checked = !completed;
        alert('Failed to update completion status');
    });
};

// Function to add a newly created board to the search modal's recent boards
function addBoardToSearchModal(boardId, boardTitle, boardDescription, boardLink) {
    const searchModal = document.getElementById('search-modal');
    if (!searchModal) return;

    // Find the Recent Boards section by looking for the heading
    const recentBoardsHeading = Array.from(searchModal.querySelectorAll('h3')).find(
        h3 => h3.textContent.includes('Recent Boards')
    );
    if (!recentBoardsHeading) return;

    // Get the container with recent boards
    const recentBoardsSection = recentBoardsHeading.nextElementSibling;
    if (!recentBoardsSection || !recentBoardsSection.classList.contains('space-y-2')) return;

    // Create the new board entry
    const newBoardEntry = document.createElement('a');
    newBoardEntry.href = boardLink;
    newBoardEntry.className = 'flex items-center px-3 py-2 text-gray-700 hover:bg-gray-100 rounded-md';
    newBoardEntry.setAttribute('onclick', 'closeSearch()');

    const innerHTML = `
        <div class="w-5 h-5 mr-3 bg-blue-500 rounded flex-shrink-0"></div>
        <div class="flex-1 min-w-0">
            <div class="font-medium truncate">${boardTitle}</div>
            ${boardDescription ? `<div class="text-sm text-gray-500 truncate">${boardDescription}</div>` : ''}
        </div>
    `;
    newBoardEntry.innerHTML = innerHTML;

    // Add the new board at the top of the recent boards list
    recentBoardsSection.insertBefore(newBoardEntry, recentBoardsSection.firstChild);

    // Keep only the 5 most recent boards
    const allBoards = recentBoardsSection.querySelectorAll('a');
    if (allBoards.length > 5) {
        for (let i = 5; i < allBoards.length; i++) {
            allBoards[i].remove();
        }
    }

    console.log('Added new board to search modal:', boardTitle);
}

// Function to update the main boards count
function updateMainBoardsCount(delta) {
    const countElement = document.getElementById('main-boards-count');
    if (countElement) {
        const currentCount = parseInt(countElement.textContent) || 0;
        const newCount = Math.max(0, currentCount + delta);
        countElement.textContent = newCount;
        console.log(`Main boards count updated: ${currentCount} -> ${newCount}`);
    }
}

// Function to update the nested boards count
function updateNestedBoardsCount(delta) {
    const countElement = document.getElementById('nested-boards-count');
    if (countElement) {
        const currentCount = parseInt(countElement.textContent) || 0;
        const newCount = Math.max(0, currentCount + delta);
        countElement.textContent = newCount;
        console.log(`Nested boards count updated: ${currentCount} -> ${newCount}`);
    }
}

// Function to update the total boards stat in the dashboard stats section
function updateTotalBoardsCount(delta) {
    const countElement = document.querySelector('[data-stat="total-boards"]');
    if (countElement) {
        const currentCount = parseInt(countElement.textContent) || 0;
        const newCount = Math.max(0, currentCount + delta);
        countElement.textContent = newCount;
        console.log(`Total boards count updated: ${currentCount} -> ${newCount}`);
    }
}

// Function to update the active tasks stat
function updateActiveTasksCount(delta) {
    const countElement = document.querySelector('[data-stat="active-tasks"]');
    if (countElement) {
        const currentCount = parseInt(countElement.textContent) || 0;
        const newCount = Math.max(0, currentCount + delta);
        countElement.textContent = newCount;
        console.log(`Active tasks count updated: ${currentCount} -> ${newCount}`);
    }
}

// Function to update the due soon stat
function updateDueSoonCount(delta) {
    const countElement = document.querySelector('[data-stat="due-soon"]');
    if (countElement) {
        const currentCount = parseInt(countElement.textContent) || 0;
        const newCount = Math.max(0, currentCount + delta);
        countElement.textContent = newCount;
        console.log(`Due soon count updated: ${currentCount} -> ${newCount}`);
    }
}

// Function to update the collaborators stat
function updateCollaboratorsCount(delta) {
    const countElement = document.querySelector('[data-stat="collaborators"]');
    if (countElement) {
        const currentCount = parseInt(countElement.textContent) || 0;
        const newCount = Math.max(0, currentCount + delta);
        countElement.textContent = newCount;
        console.log(`Collaborators count updated: ${currentCount} -> ${newCount}`);
    }
}

// Helper function to check if a deadline is "due soon" (overdue or within 7 days)
function isTaskDueSoon(deadlineStr) {
    if (!deadlineStr) return false;

    try {
        const deadline = new Date(deadlineStr);
        const now = new Date();
        const diffTime = deadline - now;
        const diffDays = diffTime / (1000 * 60 * 60 * 24);

        // Due soon if overdue (negative) or within 7 days
        return diffDays <= 7;
    } catch (e) {
        return false;
    }
}

// Helper function to extract deadline from task card or task data
function getTaskDeadline(taskCard) {
    // Try to find deadline in the task card DOM
    const deadlineElement = taskCard.querySelector('[data-deadline]');
    if (deadlineElement) {
        return deadlineElement.getAttribute('data-deadline');
    }

    // Alternative: look for deadline in visible text (Due today, Due in Xd, etc)
    const deadlineText = taskCard.querySelector('.text-red-800, .text-orange-800, .text-blue-800');
    if (deadlineText) {
        // If we see "Overdue" or "Due today" or "Due in Xd", it's likely due soon
        const text = deadlineText.textContent.trim();
        if (text.includes('Overdue') || text.includes('Due today') || text.includes('Due in')) {
            return 'soon'; // Simplified flag
        }
    }

    return null;
}

// Function to fetch and update collaborators count from server (source of truth)
function refreshCollaboratorsCount() {
    fetch('/api/dashboard/collaborators-count', {
        method: 'GET',
        credentials: 'include'
    })
    .then(response => response.json())
    .then(data => {
        if (data.count !== undefined) {
            const countElement = document.querySelector('[data-stat="collaborators"]');
            if (countElement) {
                countElement.textContent = data.count;
                console.log('Collaborators count refreshed from server:', data.count);
            }
        }
    })
    .catch(error => {
        console.error('Error fetching collaborators count:', error);
    });
}