import { dom } from './dom.js';
import { state } from './state.js';

export async function loadWorkspaces() {
    if (!window.go?.app?.App?.ListWorkspaces) return;
    try {
        state.workspacesCache = await window.go.app.App.ListWorkspaces();
    } catch (error) {
        console.error('加载工作目录失败:', error);
        state.workspacesCache = [];
    }
    syncCurrentThreadWorkspace();
}

export function updateWorkspaceToggle() {
    if (!dom.workspaceToggleLabel) return;
    const active = state.workspacesCache.find((workspace) => workspace.path === state.currentWorkspacePath);
    dom.workspaceToggleLabel.textContent = active
        ? getWorkspaceName(active.path)
        : state.currentWorkspacePath
          ? getWorkspaceName(state.currentWorkspacePath)
          : 'Workspace';
}

export function updateWorkspaceToggleState() {
    if (!dom.workspaceToggle) return;
    dom.workspaceToggle.disabled = state.isProcessing || !state.currentThreadID || state.workspacesCache.length === 0;
}

export function setWorkspaceDropdownOpen(isOpen) {
    if (!dom.workspaceSwitcher || !dom.workspaceDropdown || !dom.workspaceToggle) return;
    dom.workspaceSwitcher.classList.toggle('open', isOpen);
    dom.workspaceToggle.setAttribute('aria-expanded', String(isOpen));
    dom.workspaceDropdown.setAttribute('aria-hidden', String(!isOpen));
    if (isOpen && dom.workspaceSearchInput) {
        dom.workspaceSearchInput.focus();
    }
}

export function renderWorkspaceList() {
    if (!dom.workspaceList) return;
    dom.workspaceList.innerHTML = '';

    const term = state.workspaceSearchTerm.toLowerCase();
    const filtered = state.workspacesCache.filter(
        (w) => !term || getWorkspaceName(w.path).toLowerCase().includes(term) || w.path.toLowerCase().includes(term)
    );

    if (filtered.length === 0) {
        dom.workspaceList.innerHTML = `<div class="workspace-empty">${state.workspacesCache.length === 0 ? 'No workspaces available' : 'No workspaces found'}</div>`;
        return;
    }

    filtered.forEach((workspace) => {
        const item = document.createElement('button');
        item.type = 'button';
        item.className = 'workspace-item';
        if (workspace.path === state.currentWorkspacePath) item.classList.add('selected');
        item.innerHTML = `
            <span class="workspace-item-name">${getWorkspaceName(workspace.path)}</span>
            <span class="workspace-item-path">${workspace.path}</span>
        `;
        item.addEventListener('click', () => setWorkspace(workspace.path));
        dom.workspaceList.appendChild(item);
    });
}

async function setWorkspace(workspacePath) {
    if (
        !workspacePath ||
        !state.currentThreadID ||
        state.isProcessing ||
        workspacePath === state.currentWorkspacePath
    ) {
        setWorkspaceDropdownOpen(false);
        return;
    }
    try {
        await window.go.app.App.UpdateWorkspace(state.currentThreadID, workspacePath);
        state.currentWorkspacePath = workspacePath;
        const thread = state.threadsCache.find((item) => item.id === state.currentThreadID);
        if (thread) thread.work_dir = workspacePath;
        await loadWorkspaces();
        setWorkspaceDropdownOpen(false);
    } catch (error) {
        console.error('切换工作目录失败:', error);
    }
}

export async function selectWorkspaceDirectory() {
    if (state.isProcessing || !state.currentThreadID) return;
    try {
        const selectedPath = await window.go.app.App.SelectWorkspace(state.currentThreadID);
        if (selectedPath) {
            state.currentWorkspacePath = selectedPath;
            const thread = state.threadsCache.find((item) => item.id === state.currentThreadID);
            if (thread) thread.work_dir = selectedPath;
        }
        await loadWorkspaces();
    } catch (error) {
        console.error('选择工作目录失败:', error);
    }
}

export function syncCurrentThreadWorkspace() {
    const thread = state.threadsCache.find((item) => item.id === state.currentThreadID);
    state.currentWorkspacePath = thread?.work_dir || getDefaultWorkspacePath() || '';
    updateWorkspaceToggle();
    renderWorkspaceList();
    updateWorkspaceToggleState();
}

export function setupWorkspaceHandlers() {
    dom.workspaceSelectBtn?.addEventListener('click', async (e) => {
        e.stopPropagation();
        await selectWorkspaceDirectory();
        setWorkspaceDropdownOpen(false);
    });

    if (dom.workspaceSearchInput) {
        dom.workspaceSearchInput.addEventListener('input', () => {
            state.workspaceSearchTerm = dom.workspaceSearchInput.value.trim().toLowerCase();
            renderWorkspaceList();
        });
        dom.workspaceSearchInput.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                setWorkspaceDropdownOpen(false);
                dom.workspaceToggle?.focus();
            }
        });
    }
}

function getWorkspaceName(path) {
    return path?.split('/').filter(Boolean).pop() || path || '';
}

function getDefaultWorkspacePath() {
    const fallback = state.workspacesCache.find((w) => w.is_default);
    return fallback ? fallback.path : state.workspacesCache[0]?.path || '';
}
