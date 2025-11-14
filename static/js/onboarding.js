/**
 * SUDO Onboarding System with Driver.js
 * Smooth animations, proper element targeting, and auto-focus
 */

// ==================== STATE MANAGEMENT ====================
const OnboardingState = {
    STORAGE_KEY: 'sudo_onboarding_v2',

    get() {
        const stored = sessionStorage.getItem(this.STORAGE_KEY);
        return stored ? JSON.parse(stored) : {
            completed: false,
            currentStep: 0,
            skipped: false,
            startedAt: null,
            boardCreated: false,
            boardId: null,
            taskCreated: false
        };
    },

    set(state) {
        sessionStorage.setItem(this.STORAGE_KEY, JSON.stringify(state));
    },

    reset() {
        sessionStorage.removeItem(this.STORAGE_KEY);
    },

    isCompleted() {
        const state = this.get();
        return state.completed || state.skipped;
    },

    markCompleted() {
        const state = this.get();
        state.completed = true;
        this.set(state);

        // Persist to database
        fetch('/settings/complete-onboarding', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            }
        }).catch(err => console.error('Failed to persist onboarding completion:', err));
    },

    markCompletedLocally() {
        // Mark as completed in session storage only (without API call)
        const state = this.get();
        state.completed = true;
        this.set(state);
    },

    markSkipped() {
        const state = this.get();
        state.skipped = true;
        this.set(state);
    },

    updateStep(step) {
        const state = this.get();
        state.currentStep = step;
        this.set(state);
    }
};

// ==================== ONBOARDING CONTROLLER ====================
class OnboardingController {
    constructor() {
        this.driver = null;
        this.currentStepIndex = 0;
        this.isActive = false;
        this.cleanupHandlers = [];
        this.allowFormSubmit = false; // Flag to control form submission during tour
    }

    // Initialize Driver.js
    initDriver() {
        console.log('Initializing Driver.js...');

        if (!window.driver || !window.driver.js || typeof window.driver.js.driver !== 'function') {
            console.error('Driver.js not loaded properly!', {
                hasDriver: !!window.driver,
                hasJs: !!window.driver?.js,
                hasDriverFn: typeof window.driver?.js?.driver
            });
            return null;
        }

        console.log('Driver.js is available');

        this.driver = window.driver.js.driver({
            showProgress: true,
            showButtons: ['next', 'previous'],
            animate: true,
            overlayColor: 'rgba(0, 0, 0, 0.75)',
            overlayOpacity: 0.75,
            smoothScroll: true,
            stagePadding: 15,  // More padding around highlighted elements
            stageRadius: 10,   // Rounder corners for spotlight
            allowClose: false,
            disableActiveInteraction: false,  // Allow interaction with highlighted elements

            nextBtnText: 'Next →',
            prevBtnText: '← Back',
            doneBtnText: 'Finish',

            onDestroyStarted: () => {
                console.log('Driver destroy started');
                if (!OnboardingState.isCompleted()) {
                    if (confirm('Exit the tutorial?')) {
                        this.skip();
                        return true;
                    }
                    return false;
                }
            },

            onDestroyed: () => {
                console.log('Driver destroyed');
                this.isActive = false;
                this.cleanup();
            },

            onHighlightStarted: (element, step) => {
                console.log('Highlight starting:', {
                    element: element,
                    tagName: element?.tagName,
                    id: element?.id,
                    step: step
                });
            },

            // Auto-focus elements when highlighted
            onHighlighted: (element, step) => {
                console.log('Element highlighted:', {
                    element: element,
                    tagName: element?.tagName,
                    id: element?.id,
                    step: step
                });

                if (element) {
                    // Auto-focus inputs and textareas
                    if (element.tagName === 'INPUT' || element.tagName === 'TEXTAREA') {
                        console.log('Auto-focusing input element');
                        setTimeout(() => {
                            element.focus();
                            console.log('  Element focused');
                            // Place cursor at end of existing content
                            if (element.value) {
                                element.setSelectionRange(element.value.length, element.value.length);
                                console.log('  Cursor positioned');
                            }
                        }, 300);
                    }
                }
            }
        });

        console.log('Driver initialized:', !!this.driver);
        console.log('  Driver methods:', Object.keys(this.driver));

        return this.driver;
    }

    // Main initialization
    async init() {
        // Read onboarding status from data attribute
        const mainContainer = document.querySelector('[data-onboarding-completed]');
        const dbCompleted = mainContainer?.getAttribute('data-onboarding-completed') === 'true';

        console.log('Onboarding Init', {
            pathname: window.location.pathname,
            state: OnboardingState.get(),
            dbCompleted: dbCompleted
        });

        // Check database flag first - this persists across sessions
        if (dbCompleted === true) {
            console.log('Onboarding already completed (from database)');
            // Mark session storage as completed to avoid re-checking (no API call needed)
            OnboardingState.markCompletedLocally();
            return;
        }

        // Check session storage for current session
        if (OnboardingState.isCompleted()) {
            console.log('Onboarding already completed (from session)');
            return;
        }

        const state = OnboardingState.get();

        // Dashboard steps (0-2)
        if (window.location.pathname === '/dashboard') {
            if (!state.startedAt) {
                console.log('Showing welcome modal');
                this.showWelcomeModal();
            } else if (state.currentStep < 3 && !state.boardCreated) {
                console.log('Resuming onboarding on dashboard');
                setTimeout(() => this.resumeOnboarding(), 300);
            }
        }
        // Board steps (3-4)
        else if (window.location.pathname.startsWith('/boards/')) {
            console.log('On board page, checking state...', {
                startedAt: state.startedAt,
                currentStep: state.currentStep,
                boardCreated: state.boardCreated,
                completed: state.completed
            });

            // If we just created a board (step 3) or are on task creation (step 4)
            if (state.startedAt && state.currentStep >= 3 && !state.completed && state.boardCreated) {
                console.log('Resuming onboarding on board page - waiting for board to load...');

                // Wait longer for board columns to be in DOM
                setTimeout(() => {
                    console.log('Starting Step 3 (task creation)');
                    this.resumeOnboarding();
                }, 1000); // 1 second delay to ensure board columns are fully loaded
            }
        }
    }

    showWelcomeModal() {
        const modal = document.getElementById('welcome-modal');
        if (modal) {
            modal.classList.remove('hidden');
        }
    }

    hideWelcomeModal() {
        const modal = document.getElementById('welcome-modal');
        if (modal) {
            modal.classList.add('hidden');
        }
    }

    start() {
        console.log('Starting onboarding');

        const state = OnboardingState.get();
        state.startedAt = Date.now();
        state.currentStep = 0;
        state.skipped = false;
        OnboardingState.set(state);

        this.hideWelcomeModal();
        this.currentStepIndex = 0;
        this.isActive = true;
        this.startStep1();
    }

    resumeOnboarding() {
        const state = OnboardingState.get();
        this.currentStepIndex = state.currentStep;
        this.isActive = true;

        console.log('Resuming from step', this.currentStepIndex);

        switch (this.currentStepIndex) {
            case 0:
                this.startStep1();
                break;
            case 1:
                this.startStep1Part2();
                break;
            case 2:
                this.startStep2();
                break;
            case 3:
                this.startStep3();
                break;
            case 4:
                this.startStep4();
                break;
            default:
                console.warn('Unknown step:', this.currentStepIndex);
        }
    }

    skip() {
        console.log('Skipping onboarding');
        OnboardingState.markSkipped();

        if (this.driver) {
            this.driver.destroy();
        }

        this.hideWelcomeModal();
        this.isActive = false;
        this.cleanup();
    }

    complete() {
        console.log('Onboarding completed 🎉');
        OnboardingState.markCompleted();

        this.isActive = false;
        this.showCompletionModal();
        this.celebrate();
        this.cleanup();
    }

    showCompletionModal() {
        const modal = document.getElementById('completion-modal');
        if (modal) {
            modal.classList.remove('hidden');
        }
    }

    celebrate() {
        if (window.confetti) {
            const duration = 3000;
            const end = Date.now() + duration;

            (function frame() {
                confetti({
                    particleCount: 3,
                    angle: 60,
                    spread: 55,
                    origin: { x: 0 },
                    colors: ['#3b82f6', '#60a5fa', '#93c5fd', '#10b981', '#34d399']
                });
                confetti({
                    particleCount: 3,
                    angle: 120,
                    spread: 55,
                    origin: { x: 1 },
                    colors: ['#3b82f6', '#60a5fa', '#93c5fd', '#10b981', '#34d399']
                });

                if (Date.now() < end) {
                    requestAnimationFrame(frame);
                }
            })();
        }
    }

    cleanup() {
        if (this.cleanupHandlers) {
            this.cleanupHandlers.forEach(fn => fn());
            this.cleanupHandlers = [];
        }
    }

    addCleanup(fn) {
        this.cleanupHandlers.push(fn);
    }

    // ==================== STEP 1: GLOBAL SEARCH → CREATE BOARD ====================
    startStep1() {
        console.log('Step 1: Open global search');

        OnboardingState.updateStep(0);

        if (!this.driver) {
            this.initDriver();
        }

        // Highlight search bar
        this.driver.highlight({
            element: '#search-bar-container',
            popover: {
                title: 'Step 1: Quick Actions',
                description: 'Click the search bar or press <kbd>Ctrl+K</kbd> to open the quick actions menu.',
                side: 'bottom',
                align: 'center'
            }
        });

        // Listen for search modal opening
        const searchModal = document.getElementById('search-modal');
        if (searchModal) {
            const observer = new MutationObserver((mutations) => {
                for (const mutation of mutations) {
                    if (mutation.attributeName === 'class') {
                        if (!searchModal.classList.contains('hidden')) {
                            console.log('Search modal opened by user');
                            observer.disconnect();
                            this.driver.destroy();
                            setTimeout(() => this.startStep1Part2(), 300);
                            break;
                        }
                    }
                }
            });

            observer.observe(searchModal, { attributes: true });
            this.addCleanup(() => observer.disconnect());
        }
    }

    startStep1Part2() {
        console.log('Step 1 Part 2: Click Create Board');

        OnboardingState.updateStep(1);

        const createBoardBtn = document.getElementById('search-create-board-btn');
        if (!createBoardBtn) {
            console.error('Create board button not found!');
            return;
        }

        if (!this.driver || this.driver.isDestroyed) {
            this.initDriver();
        }

        // Highlight "Create new board" button
        this.driver.highlight({
            element: '#search-create-board-btn',
            popover: {
                title: 'Step 2: Create Your First Board',
                description: 'Click "Create new board" to continue.',
                side: 'right',
                align: 'start'
            }
        });

        // Listen for button click
        const clickHandler = () => {
            console.log('Create board button clicked');
            this.driver.destroy();
            this.waitForBoardModal();
        };

        createBoardBtn.addEventListener('click', clickHandler, { once: true });
        this.addCleanup(() => createBoardBtn.removeEventListener('click', clickHandler));
    }

    waitForBoardModal() {
        console.log('Waiting for board modal...');

        const createBoardModal = document.getElementById('create-board-modal');
        if (!createBoardModal) {
            console.error('Create board modal not found!');
            return;
        }

        console.log('Board modal element found:', createBoardModal);
        console.log('  Modal classes:', createBoardModal.className);
        console.log('  Is hidden?', createBoardModal.classList.contains('hidden'));

        // Close search modal
        const searchModal = document.getElementById('search-modal');
        if (searchModal) {
            searchModal.classList.add('hidden');
            console.log('Search modal closed');
        }

        // Check if modal is already open
        if (!createBoardModal.classList.contains('hidden')) {
            console.log('Modal is already open, starting Step 2');
            setTimeout(() => this.startStep2(), 400);
            return;
        }

        // Watch for modal opening
        console.log('Setting up observer for modal...');
        const observer = new MutationObserver((mutations) => {
            console.log('Modal mutation detected:', mutations.length, 'mutations');
            for (const mutation of mutations) {
                console.log('  Mutation type:', mutation.attributeName);
                console.log('  New classes:', createBoardModal.className);

                if (mutation.attributeName === 'class') {
                    const isHidden = createBoardModal.classList.contains('hidden');
                    console.log('  Is hidden now?', isHidden);

                    if (!isHidden) {
                        console.log('Board creation modal opened!');
                        observer.disconnect();
                        setTimeout(() => this.startStep2(), 400);
                        break;
                    }
                }
            }
        });

        observer.observe(createBoardModal, { attributes: true });
        this.addCleanup(() => observer.disconnect());
    }

    // ==================== STEP 2: BOARD CREATION FORM ====================
    startStep2() {
        console.log('Step 2: Fill board details');

        OnboardingState.updateStep(2);

        console.log('Checking driver state...');
        console.log('  this.driver exists?', !!this.driver);
        console.log('  this.driver.isDestroyed?', this.driver?.isDestroyed);

        if (!this.driver || this.driver.isDestroyed) {
            console.log('  Re-initializing driver...');
            this.initDriver();
            console.log('  Driver initialized:', !!this.driver);
        }

        // Wait a bit for modal DOM to settle
        setTimeout(() => {
            const titleInput = document.getElementById('board-title-input');
            const descriptionInput = document.getElementById('board-description') ||
                                    document.querySelector('#create-board-modal textarea[name="description"]');
            const form = document.querySelector('#create-board-modal form');
            const submitButton = form?.querySelector('button[type="submit"]');

            console.log('Form elements:');
            console.log('  Title input:', titleInput);
            console.log('  Description:', descriptionInput);
            console.log('  Submit button:', submitButton);
            console.log('  Form:', form);

            if (!titleInput || !form || !submitButton) {
                console.error('Form elements not found!');
                return;
            }

            // Modify form for full page navigation FIRST (before setting up steps)
            // IMPORTANT: Must disable HTMX completely or it will still intercept the form!
            form.removeAttribute('hx-post');
            form.removeAttribute('hx-target');
            form.removeAttribute('hx-swap');
            form.removeAttribute('hx-on::after-request');

            // Remove HTMX's internal data completely
            if (form['htmx-internal-data']) {
                delete form['htmx-internal-data'];
                console.log('Removed HTMX internal data');
            }

            // Set standard form attributes for normal POST
            form.setAttribute('action', '/boards');
            form.setAttribute('method', 'POST');

            console.log('Form configured for full page navigation');

            // Create steps for all form fields including submit button
            const formSteps = [
                {
                    element: titleInput,
                    popover: {
                        title: 'Step 3: Board Name',
                        description: 'Enter a name for your board (e.g., "My First Project"). This field is required.',
                        side: 'bottom',
                        align: 'start'
                    }
                }
            ];

            // Add description field if it exists
            if (descriptionInput) {
                formSteps.push({
                    element: descriptionInput,
                    popover: {
                        title: 'Step 4: Description (Optional)',
                        description: 'Add a description to remember what this board is for. Press Tab or click Next to continue.',
                        side: 'bottom',
                        align: 'start'
                    }
                });
            }

            // Add submit button as final step - user can click button directly or use Next
            formSteps.push({
                element: submitButton,
                popover: {
                    title: 'Step 5: Create Your Board',
                    description: 'Click "Create Board" below to create your board and continue!',
                    side: 'top',
                    align: 'center',
                    showButtons: [], // No buttons - user must click the actual button
                }
            });

            // Ensure button click immediately submits the form
            const handleButtonClick = (e) => {
                console.log('Create Board button clicked!');

                // Prevent default to stop any HTMX interception
                e.preventDefault();
                e.stopPropagation();
                e.stopImmediatePropagation();

                // Update state immediately
                OnboardingState.updateStep(3);
                const state = OnboardingState.get();
                state.boardCreated = true;
                OnboardingState.set(state);

                console.log('State updated, submitting form programmatically');

                // Use the native HTMLFormElement.submit() method
                // This bypasses all event listeners including HTMX
                HTMLFormElement.prototype.submit.call(form);
            };

            submitButton.addEventListener('click', handleButtonClick, { capture: true, once: true });
            this.addCleanup(() => submitButton.removeEventListener('click', handleButtonClick, { capture: true }));

            console.log('Form steps created:', formSteps.length, 'steps');
            console.log('  Steps:', formSteps);

            // Don't prevent form submission - let users click the button directly
            // Just handle it in the onNextClick callback for the Next button
            console.log('Form submission enabled - users can click Create Board directly');

            // Start the tour
            console.log('Starting Driver.js tour...');

            try {
                this.driver.setSteps(formSteps);
                console.log('  Steps set');

                this.driver.drive();
                console.log('  Drive started');
            } catch (error) {
                console.error('Error starting driver:', error);
            }
        }, 200); // Wait 200ms for modal DOM to settle
    }

    // ==================== STEP 3: CREATE TASK ====================
    startStep3() {
        console.log('Step 3: Create first task');

        OnboardingState.updateStep(3);

        console.log('Looking for board columns...');

        // Wait for board columns to load
        this.waitForBoardColumns().then(() => {
            console.log('Board columns loaded, finding first column...');

            const boardColumnsContainer = document.getElementById('board-columns');
            const columns = document.querySelectorAll('#board-columns .kanban-column');
            const firstColumn = columns[0];
            const addTaskBtn = firstColumn?.querySelector('.add-task-btn');

            console.log('Board layout:', {
                columnsContainer: !!boardColumnsContainer,
                columnsCount: columns.length,
                firstColumn: !!firstColumn,
                addTaskBtn: !!addTaskBtn
            });

            if (!addTaskBtn) {
                console.error('Add task button not found!');
                console.log('  First column HTML:', firstColumn?.innerHTML.substring(0, 200));
                return;
            }

            console.log('Add task button found:', addTaskBtn);

            if (!this.driver || this.driver.isDestroyed) {
                console.log('Re-initializing driver...');
                this.initDriver();
            }

            // Scroll into view
            console.log('Scrolling column into view...');
            firstColumn.scrollIntoView({ behavior: 'smooth', block: 'nearest' });

            // Highlight add task button
            this.driver.highlight({
                element: addTaskBtn,
                popover: {
                    title: 'Step 6: Create Your First Task',
                    description: 'Click the + button to add a task in the To-Do column.',
                    side: 'bottom',
                    align: 'center'
                }
            });

            // Listen for button click
            const clickHandler = () => {
                console.log('Add task button clicked');
                this.driver.destroy();
                this.waitForTaskForm(firstColumn);
            };

            addTaskBtn.addEventListener('click', clickHandler, { once: true });
            this.addCleanup(() => addTaskBtn.removeEventListener('click', clickHandler));
        });
    }

    waitForBoardColumns() {
        return new Promise((resolve) => {
            const checkColumns = () => {
                const columns = document.querySelectorAll('#board-columns .kanban-column');
                if (columns.length > 0) {
                    console.log('Board columns loaded');
                    resolve();
                } else {
                    setTimeout(checkColumns, 100);
                }
            };
            checkColumns();
        });
    }

    waitForTaskForm(column) {
        const checkForm = () => {
            const form = column.querySelector('form');
            const titleInput = column.querySelector('.task-title-input');

            if (form && titleInput && !form.parentElement.classList.contains('hidden')) {
                console.log('Task form appeared');
                setTimeout(() => this.startStep4(form, column), 400);
            } else {
                setTimeout(checkForm, 100);
            }
        };

        setTimeout(checkForm, 100);
    }

    // ==================== STEP 4: TASK FORM FIELDS ====================
    startStep4(form, column) {
        console.log('Step 4: Fill task details');

        OnboardingState.updateStep(4);

        if (!this.driver || this.driver.isDestroyed) {
            this.initDriver();
        }

        // Get form fields
        const titleInput = form.querySelector('.task-title-input') || form.querySelector('input[name="title"]');
        const assigneeCheckboxes = form.querySelectorAll('input[name="assignee_ids[]"]');
        const firstAssigneeCheckbox = assigneeCheckboxes[0]; // Get first checkbox
        const assigneeContainer = firstAssigneeCheckbox?.closest('div.mb-3'); // Get the container div
        const descriptionInput = form.querySelector('textarea[name="description"]');
        const prioritySelect = form.querySelector('select[name="priority"]');
        const deadlineInput = form.querySelector('input[name="deadline"]');
        const tagsInput = form.querySelector('input[name="tags"]');

        console.log('Task form fields found:', {
            titleInput: !!titleInput,
            assigneeCheckboxes: assigneeCheckboxes.length,
            firstAssigneeCheckbox: !!firstAssigneeCheckbox,
            assigneeContainer: !!assigneeContainer,
            descriptionInput: !!descriptionInput,
            prioritySelect: !!prioritySelect,
            deadlineInput: !!deadlineInput,
            tagsInput: !!tagsInput
        });

        // Build steps for available fields
        const taskFormSteps = [];

        if (titleInput) {
            taskFormSteps.push({
                element: titleInput,
                popover: {
                    title: 'Step 7: Task Title',
                    description: 'Enter a title for your task (required). For example: "Setup project repository"',
                    side: 'top',
                    align: 'start'
                }
            });
        }

        if (descriptionInput) {
            taskFormSteps.push({
                element: descriptionInput,
                popover: {
                    title: 'Step 8: Description',
                    description: 'Add task details (optional). You can Tab, click, or press Enter to navigate.',
                    side: 'top',
                    align: 'start'
                }
            });
        }

        if (prioritySelect) {
            taskFormSteps.push({
                element: prioritySelect,
                popover: {
                    title: 'Step 9: Priority',
                    description: 'Set the task priority level (optional).',
                    side: 'top',
                    align: 'start'
                }
            });
        }

        if (deadlineInput) {
            taskFormSteps.push({
                element: deadlineInput,
                popover: {
                    title: 'Step 10: Deadline',
                    description: 'Set a due date for this task (optional).',
                    side: 'top',
                    align: 'start'
                }
            });
        }

        // Add assignee step after deadline (required field)
        if (assigneeContainer) {
            taskFormSteps.push({
                element: assigneeContainer,
                popover: {
                    title: 'Step 11: Assign Task',
                    description: 'Select at least one person to assign this task to (required). Check the box next to a name.',
                    side: 'top',
                    align: 'start'
                }
            });
        }

        if (tagsInput) {
            taskFormSteps.push({
                element: tagsInput,
                popover: {
                    title: 'Step 12: Tags',
                    description: 'Add tags to organize your tasks (optional). Separate with commas.',
                    side: 'top',
                    align: 'start'
                }
            });
        }

        // Add final step: Create Task button
        const submitButton = form.querySelector('button[type="submit"]');
        if (submitButton) {
            taskFormSteps.push({
                element: submitButton,
                popover: {
                    title: 'Step 13: Create Your First Task',
                    description: 'Now click "Add Task" to save your first task to the board!',
                    side: 'top',
                    align: 'center'
                }
            });
        }

        // Start the task form tour
        this.driver.setSteps(taskFormSteps);
        this.driver.drive();

        // Listen for form submission
        const submitHandler = (e) => {
            console.log('Task form submitted!');

            // Destroy the driver immediately so user sees the full board
            if (this.driver) {
                this.driver.destroy();
            }

            // Wait for HTMX to create the task and add it to the DOM
            this.waitForTaskCreation(column);
        };

        form.addEventListener('submit', submitHandler, { once: true });
        this.addCleanup(() => form.removeEventListener('submit', submitHandler));
    }

    // Wait for task card to appear after creation
    waitForTaskCreation(column) {
        console.log('Waiting for task to be created');

        let attempts = 0;
        const maxAttempts = 30;

        const checkForTask = () => {
            const taskCards = column.querySelectorAll('.task-card');

            if (taskCards.length > 0) {
                console.log('Task card appeared, completing onboarding');
                OnboardingState.updateStep(5);
                setTimeout(() => this.complete(), 300);
            } else if (attempts < maxAttempts) {
                attempts++;
                setTimeout(checkForTask, 100);
            } else {
                console.log('Task creation timeout, completing anyway');
                this.complete();
            }
        };

        setTimeout(checkForTask, 200);
    }
}

// ==================== GLOBAL INSTANCE ====================
let onboardingController = null;

// ==================== INITIALIZATION ====================
document.addEventListener('DOMContentLoaded', () => {
    console.log('DOM Content Loaded - Initializing onboarding');
    onboardingController = new OnboardingController();
    window.onboardingController = onboardingController;
    onboardingController.init();
});

// HTMX integration
document.body.addEventListener('htmx:afterSettle', (event) => {
    console.log('HTMX content swapped');
    if (onboardingController && !OnboardingState.isCompleted()) {
        setTimeout(() => onboardingController.init(), 100);
    }
});

// ==================== GLOBAL FUNCTIONS ====================
window.startOnboarding = function() {
    if (onboardingController) {
        onboardingController.start();
    }
};

window.skipOnboarding = function() {
    if (onboardingController) {
        onboardingController.skip();
    }
};

window.closeCompletionModal = function() {
    const modal = document.getElementById('completion-modal');
    if (modal) {
        modal.classList.add('hidden');
    }
};

window.restartOnboarding = function() {
    OnboardingState.reset();
    window.location.href = '/dashboard';
};

// Keyboard shortcut: Ctrl+Shift+H to restart
document.addEventListener('keydown', (e) => {
    if (e.ctrlKey && e.shiftKey && e.key === 'H') {
        e.preventDefault();
        if (confirm('Restart onboarding tutorial?')) {
            window.restartOnboarding();
        }
    }
});

// Debug
console.log('Onboarding system loaded');
console.log('Driver.js:', {
    available: !!(window.driver && window.driver.js && window.driver.js.driver)
});
