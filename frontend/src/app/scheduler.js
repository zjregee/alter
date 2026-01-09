import { formatFeedTimestamp } from './utils.js';

let schedulesCache = [];
let currentScheduleId = null;
let isEditing = false;
let activePollTimeout = null;

export function setupSchedulerHandlers() {
    // Setup global handlers if needed
}

export async function loadSchedules() {
    const container = document.querySelector('.scheduler-container');
    if (!container) return;

    if (!window.go?.app?.App?.ListSchedules) {
        container.innerHTML = '<div class="scheduler-empty">Scheduler API not available</div>';
        return;
    }

    // Ensure modal exists
    ensureRunDetailModal();

    try {
        const schedules = await window.go.app.App.ListSchedules();
        schedulesCache = schedules || [];

        // Ensure container structure exists
        if (!container.querySelector('.scheduler-sidebar')) {
            renderSchedulerLayout(container);
        }

        // Select first schedule if none selected and schedules exist
        if (!currentScheduleId && schedulesCache.length > 0) {
            currentScheduleId = schedulesCache[0].id;
        } else if (currentScheduleId && !schedulesCache.find((s) => s.id === currentScheduleId)) {
            // If currently selected schedule no longer exists, select the first one or null
            currentScheduleId = schedulesCache.length > 0 ? schedulesCache[0].id : null;
        }

        renderSchedulerSidebar();
        renderSchedulerDetail();
    } catch (error) {
        console.error('Failed to load schedules:', error);
        container.innerHTML = `<div class="scheduler-empty">Error loading schedules: ${error}</div>`;
    }
}

function ensureRunDetailModal() {
    if (document.getElementById('scheduler-run-modal')) return;

    const modal = document.createElement('div');
    modal.id = 'scheduler-run-modal';
    modal.className = 'scheduler-modal-overlay';
    modal.innerHTML = `
        <div class="scheduler-modal">
            <div class="scheduler-modal-header">
                <div class="scheduler-modal-title">Run Details</div>
                <div class="scheduler-modal-nav">
                    <button class="scheduler-modal-nav-btn" id="modal-nav-prev" title="Newer">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="18 15 12 9 6 15"></polyline>
                        </svg>
                    </button>
                    <button class="scheduler-modal-nav-btn" id="modal-nav-next" title="Older">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="6 9 12 15 18 9"></polyline>
                        </svg>
                    </button>
                </div>
                <button class="scheduler-modal-close">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <line x1="18" y1="6" x2="6" y2="18"></line>
                        <line x1="6" y1="6" x2="18" y2="18"></line>
                    </svg>
                </button>
            </div>
            <div class="scheduler-modal-body" id="scheduler-modal-content">
                <!-- Content injected here -->
            </div>
        </div>
    `;

    document.body.appendChild(modal);

    const closeBtn = modal.querySelector('.scheduler-modal-close');
    closeBtn.onclick = closeRunDetail;
    modal.onclick = (e) => {
        if (e.target === modal) closeRunDetail();
    };

    // Close on Escape
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && modal.classList.contains('active')) {
            closeRunDetail();
        }
    });
}

function showRunDetail(run, allRuns) {
    const modal = document.getElementById('scheduler-run-modal');
    const content = document.getElementById('scheduler-modal-content');
    if (!modal || !content) return;

    const duration = run.ended_at
        ? ((new Date(run.ended_at) - new Date(run.started_at)) / 1000).toFixed(3) + 's'
        : 'Running...';

    const statusColorClass = `status-${run.status}`;

    content.innerHTML = `
        <div class="scheduler-modal-row">
            <div class="scheduler-modal-label">Run ID</div>
            <div class="scheduler-modal-value">${run.id}</div>
        </div>
        <div class="scheduler-modal-row">
            <div class="scheduler-modal-label">Status</div>
            <div class="scheduler-modal-value ${statusColorClass}">${run.status}</div>
        </div>
        <div class="scheduler-modal-row">
            <div class="scheduler-modal-label">Started At</div>
            <div class="scheduler-modal-value">${run.started_at ? new Date(run.started_at).toLocaleString() : '-'}</div>
        </div>
         <div class="scheduler-modal-row">
            <div class="scheduler-modal-label">Ended At</div>
            <div class="scheduler-modal-value">${run.ended_at ? new Date(run.ended_at).toLocaleString() : '-'}</div>
        </div>
        <div class="scheduler-modal-row">
            <div class="scheduler-modal-label">Duration</div>
            <div class="scheduler-modal-value">${duration}</div>
        </div>
        <div class="scheduler-modal-row">
            <div class="scheduler-modal-label">Error Output</div>
            <div class="scheduler-modal-value" style="min-height: 60px; color: ${run.error ? 'var(--error-text)' : 'var(--text-tertiary)'}">${run.error || 'No error'}</div>
        </div>
    `;

    // Navigation logic
    const prevBtn = document.getElementById('modal-nav-prev');
    const nextBtn = document.getElementById('modal-nav-next');

    if (allRuns && allRuns.length > 0) {
        const index = allRuns.findIndex((r) => r.id === run.id);

        // Prev = Newer (Lower index)
        if (index > 0) {
            prevBtn.disabled = false;
            prevBtn.onclick = () => showRunDetail(allRuns[index - 1], allRuns);
        } else {
            prevBtn.disabled = true;
            prevBtn.onclick = null;
        }

        // Next = Older (Higher index)
        if (index < allRuns.length - 1 && index !== -1) {
            nextBtn.disabled = false;
            nextBtn.onclick = () => showRunDetail(allRuns[index + 1], allRuns);
        } else {
            nextBtn.disabled = true;
            nextBtn.onclick = null;
        }
    } else {
        prevBtn.disabled = true;
        nextBtn.disabled = true;
    }

    modal.classList.add('active');
}

function closeRunDetail() {
    const modal = document.getElementById('scheduler-run-modal');
    if (modal) modal.classList.remove('active');
}

function renderSchedulerLayout(container) {
    container.innerHTML = `
        <div class="scheduler-sidebar">
            <div class="scheduler-list"></div>
        </div>
        <div class="scheduler-main">
            <div class="scheduler-detail-view"></div>
        </div>
    `;
}

function renderSchedulerSidebar() {
    const list = document.querySelector('.scheduler-list');
    if (!list) return;

    list.innerHTML = '';

    if (schedulesCache.length === 0) {
        list.innerHTML = '<div class="scheduler-empty" style="padding: 12px;">No tasks</div>';
        return;
    }

    schedulesCache.forEach((schedule) => {
        const item = document.createElement('div');
        item.className = `scheduler-list-item ${schedule.id === currentScheduleId ? 'active' : ''}`;

        item.innerHTML = `
            <div class="scheduler-list-item-name">${schedule.name || 'Untitled Task'}</div>
            <div class="scheduler-list-item-meta">
                <span>${schedule.cron_expr}</span>
                <div class="scheduler-status-dot ${schedule.enabled ? 'enabled' : ''}"></div>
            </div>
        `;

        item.onclick = () => {
            if (currentScheduleId !== schedule.id) {
                currentScheduleId = schedule.id;
                isEditing = false;
                renderSchedulerSidebar();
                renderSchedulerDetail();
            }
        };

        list.appendChild(item);
    });
}

function renderSchedulerDetail() {
    const container = document.querySelector('.scheduler-detail-view');
    if (!container) return;

    container.innerHTML = '';

    const schedule = schedulesCache.find((s) => s.id === currentScheduleId);

    if (!schedule) {
        container.innerHTML = '<div class="scheduler-empty">Select a task to view details</div>';
        return;
    }

    if (isEditing) {
        renderEditForm(container, schedule);
        return;
    }

    const wf = schedule.workflow_config || {};
    const nextRun = schedule.next_run_at ? formatFeedTimestamp(schedule.next_run_at) : 'Not scheduled';

    container.innerHTML = `
        <div class="scheduler-main-header">
            <div class="scheduler-header-top">
                <div class="scheduler-detail-name">
                    ${schedule.name || 'Untitled Task'}
                    <span class="scheduler-detail-status ${schedule.enabled ? 'enabled' : ''}">
                        ${schedule.enabled ? 'Active' : 'Disabled'}
                    </span>
                </div>
                <div class="scheduler-actions">
                    <button class="scheduler-action-btn" id="toggle-schedule-btn">
                        ${schedule.enabled ? 'Disable' : 'Enable'}
                    </button>
                    <button class="scheduler-action-btn" id="edit-schedule-btn">
                        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path></svg>
                        Edit
                    </button>
                    <button class="scheduler-action-btn primary" id="run-schedule-btn">
                        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"></polygon></svg>
                        Run Now
                    </button>
                </div>
            </div>
        </div>

        <div class="scheduler-content-body">
            <div class="scheduler-section">
                <div class="scheduler-section-title">Workflow Configuration</div>
                <div class="scheduler-config-grid">
                    <div class="scheduler-config-item">
                        <div class="config-label">Model ID</div>
                        <div class="config-value code">${wf.model_id || 'default'}</div>
                    </div>
                    <div class="scheduler-config-item">
                        <div class="config-label">Cron Schedule</div>
                        <div class="config-value code">${schedule.cron_expr}</div>
                    </div>
                    <div class="scheduler-config-item">
                        <div class="config-label">Work Directory</div>
                        <div class="config-value code">${wf.work_dir || 'default'}</div>
                    </div>
                </div>
                <div class="scheduler-prompt-block">
                    <div class="config-label">Prompt</div>
                    <div class="scheduler-prompt-content">${wf.prompt || ''}</div>
                </div>
            </div>

            <div class="scheduler-section history-wrapper">
                <div class="scheduler-section-title">Run History</div>
                <div class="scheduler-history-list">
                    <div class="scheduler-empty" style="padding: 12px;">Loading history...</div>
                </div>
            </div>

            <div class="run-timers">
                <div class="run-timer-item">
                    <span class="timer-label">Last Run:</span>
                    <span class="timer-value" id="schedule-last-run">Checking...</span>
                </div>
                <div class="run-timer-item">
                    <span class="timer-label">Next Run:</span>
                    <span class="timer-value">${nextRun}</span>
                </div>
            </div>
        </div>
    `;

    // Bind actions
    container.querySelector('#toggle-schedule-btn').onclick = () => handleToggle(schedule.id, !schedule.enabled);
    container.querySelector('#edit-schedule-btn').onclick = () => {
        isEditing = true;
        renderSchedulerDetail();
    };
    container.querySelector('#run-schedule-btn').onclick = (e) => handleTrigger(schedule.id, e.currentTarget);

    loadRunHistory(schedule.id, container.querySelector('.scheduler-history-list'));
}

function renderEditForm(container, schedule) {
    const wf = schedule.workflow_config || {};

    container.innerHTML = `
        <div class="scheduler-main-header">
            <div class="scheduler-title-group">
                <div class="scheduler-detail-name">Edit: ${schedule.name}</div>
            </div>
        </div>
        
        <div class="scheduler-edit-form">
            <div class="edit-form-row">
                <div class="edit-form-group">
                    <label class="edit-form-label">Name</label>
                    <input type="text" class="edit-form-input" id="edit-name" value="${schedule.name || ''}">
                </div>
                <div class="edit-form-group">
                    <label class="edit-form-label">Cron Expression</label>
                    <input type="text" class="edit-form-input" id="edit-cron" value="${schedule.cron_expr || ''}">
                </div>
            </div>
            
            <div class="edit-form-row">
                 <div class="edit-form-group">
                    <label class="edit-form-label">Model ID</label>
                    <input type="text" class="edit-form-input" id="edit-model" value="${wf.model_id || ''}" placeholder="Leave empty for default">
                </div>
                 <div class="edit-form-group">
                    <label class="edit-form-label">Work Directory</label>
                    <input type="text" class="edit-form-input" id="edit-workdir" value="${wf.work_dir || ''}" placeholder="Absolute path">
                </div>
            </div>

            <div class="edit-form-group">
                <label class="edit-form-label">Prompt</label>
                <textarea class="edit-form-textarea" id="edit-prompt" placeholder="Describe the task...">${wf.prompt || ''}</textarea>
            </div>

            <div class="edit-form-actions">
                <button class="scheduler-action-btn" id="cancel-edit-btn">Cancel</button>
                <button class="scheduler-action-btn primary" id="save-schedule-btn">Save Changes</button>
            </div>
        </div>
    `;

    container.querySelector('#cancel-edit-btn').onclick = () => {
        isEditing = false;
        renderSchedulerDetail();
    };
    container.querySelector('#save-schedule-btn').onclick = () => handleUpdate(schedule);
}

async function loadRunHistory(scheduleId, container) {
    if (activePollTimeout) {
        clearTimeout(activePollTimeout);
        activePollTimeout = null;
    }

    if (!container || !window.go?.app?.App?.ListScheduleRuns) {
        container.innerHTML = '<div class="scheduler-empty" style="padding: 12px;">History not available</div>';
        const lastRunEl = document.querySelector('#schedule-last-run');
        if (lastRunEl) lastRunEl.textContent = 'N/A';
        return;
    }

    try {
        const runs = await window.go.app.App.ListScheduleRuns(scheduleId);

        const lastRunEl = document.querySelector('#schedule-last-run');
        if (runs && runs.length > 0) {
            // Sort by start time descending
            runs.sort((a, b) => new Date(b.started_at) - new Date(a.started_at));
            if (lastRunEl) lastRunEl.textContent = formatFeedTimestamp(runs[0].started_at);
        } else {
            if (lastRunEl) lastRunEl.textContent = 'None';
        }

        if (!runs || runs.length === 0) {
            container.innerHTML = '<div class="scheduler-empty" style="padding: 12px;">No runs recorded</div>';
            return;
        }

        container.innerHTML = '';
        runs.forEach((run) => {
            const runEl = document.createElement('div');
            runEl.className = 'history-item';

            let statusClass = run.status; // running, finished, failed, pending
            const isRunning = statusClass === 'running' || statusClass === 'pending';

            const durationContent = run.ended_at
                ? ((new Date(run.ended_at) - new Date(run.started_at)) / 1000).toFixed(1) + 's'
                : isRunning
                  ? `
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="spinning" style="opacity: 0.6;">
                        <path d="M12 2v4m0 12v4M4.93 4.93l2.83 2.83m8.48 8.48l2.83 2.83M2 12h4m12 0h4M4.93 19.07l2.83-2.83m8.48-8.48l2.83-2.83"></path>
                    </svg>
                `
                  : '...';

            runEl.innerHTML = `
                <div class="history-status ${statusClass}" title="${run.status}"></div>
                <div class="history-time">${formatFeedTimestamp(run.started_at)}</div>
                <div class="${run.error ? 'history-error' : 'history-info'}">
                    ${run.error ? run.error : ''}
                </div>
                <div class="history-duration" style="display: flex; justify-content: flex-end; align-items: center; min-width: 40px;">
                    ${durationContent}
                </div>
            `;

            // Updated: Pass 'runs' list to showRunDetail for navigation
            runEl.onclick = () => showRunDetail(run, runs);

            container.appendChild(runEl);
        });

        // Poll if latest run is active (running or pending)
        const latestRun = runs[0];
        if (latestRun && (latestRun.status === 'running' || latestRun.status === 'pending')) {
            activePollTimeout = setTimeout(() => {
                // Ensure we are still viewing the same schedule
                if (currentScheduleId === scheduleId) {
                    const newContainer = document.querySelector('.scheduler-history-list');
                    if (newContainer) {
                        loadRunHistory(scheduleId, newContainer);
                    }
                }
            }, 1000);
        }
    } catch (error) {
        console.error('Failed to load runs:', error);
        container.innerHTML = `<div class="scheduler-empty" style="padding: 12px;">Error: ${error}</div>`;
        const lastRunEl = document.querySelector('#schedule-last-run');
        if (lastRunEl) lastRunEl.textContent = 'Error';
    }
}

async function handleToggle(id, enabled) {
    try {
        if (enabled) {
            await window.go.app.App.EnableSchedule(id);
        } else {
            await window.go.app.App.DisableSchedule(id);
        }
        await loadSchedules();
    } catch (error) {
        console.error('Failed to toggle schedule:', error);
    }
}

async function handleTrigger(id, btn) {
    // if (!confirm('Run this task immediately?')) return;

    const originalText = btn ? btn.innerHTML : '';
    if (btn) {
        btn.disabled = true;
        btn.innerHTML = `
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="spinning">
                <path d="M12 2v4m0 12v4M4.93 4.93l2.83 2.83m8.48 8.48l2.83 2.83M2 12h4m12 0h4M4.93 19.07l2.83-2.83m8.48-8.48l2.83-2.83"></path>
            </svg>
            Starting...
        `;
        // Add minimal style for spinning if not exists, or just let it be static
        btn.style.opacity = '0.7';
    }

    try {
        if (!window.go?.app?.App?.TriggerSchedule) {
            throw new Error('TriggerSchedule function not found. Please restart the app.');
        }
        await window.go.app.App.TriggerSchedule(id);

        // Refresh details after a short delay to potentially see the new run
        setTimeout(() => {
            // Only reload if we are still on the same schedule
            if (currentScheduleId === id) {
                const container = document.querySelector('.scheduler-history-list');
                if (container) loadRunHistory(id, container);
            }
            if (btn) {
                btn.disabled = false;
                btn.innerHTML = originalText;
                btn.style.opacity = '1';
            }
        }, 1000);
    } catch (error) {
        console.error('Failed to trigger schedule:', error);
        alert('Failed to trigger task: ' + error);
        if (btn) {
            btn.disabled = false;
            btn.innerHTML = originalText;
            btn.style.opacity = '1';
        }
    }
}

async function handleUpdate(originalSchedule) {
    const name = document.getElementById('edit-name').value;
    const cron = document.getElementById('edit-cron').value;
    const model = document.getElementById('edit-model').value;
    const workdir = document.getElementById('edit-workdir').value;
    const prompt = document.getElementById('edit-prompt').value;

    const updated = {
        ...originalSchedule,
        name: name,
        cron_expr: cron,
        workflow_config: {
            ...originalSchedule.workflow_config,
            model_id: model,
            work_dir: workdir,
            prompt: prompt
        }
    };

    try {
        await window.go.app.App.UpdateSchedule(originalSchedule.id, updated);
        isEditing = false;
        await loadSchedules();
    } catch (error) {
        console.error('Failed to update schedule:', error);
        alert('Failed to update schedule: ' + error);
    }
}
