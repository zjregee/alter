import { dom } from './dom.js';
import { state } from './state.js';
import { handleFeedItemPushed, setupFeedHandlers } from './feed.js';
import { setupSidebarResizer } from './layout.js';
import { setThreadRefresher, handleAgentMessage } from './chat/agent.js';
import { setupActionMenuListeners } from './chat/actions.js';
import { setupChatInputHandlers } from './chat/input.js';
import { loadModels, setModelDropdownOpen, setupModelHandlers } from './models.js';
import { loadWorkspaces, setWorkspaceDropdownOpen, setupWorkspaceHandlers } from './workspaces.js';
import {
    createNewThread,
    handleMessagesTruncated,
    handleThreadTitleUpdated,
    loadThreads,
    setupThreadUI
} from './threads.js';
import { setupViewSwitcher } from './view.js';
import { setupSettingsNavigation } from './settings.js';

export function initializeApp() {
    setThreadRefresher(() => loadThreads(false));

    setupChatInputHandlers();
    setupThreadUI();
    setupWorkspaceHandlers();
    setupModelHandlers();
    setupActionMenuListeners();
    setupSidebarResizer();
    setupFeedHandlers();
    setupSettingsNavigation();

    setupWorkspaceToggle();
    setupModelToggle();
    setupDropdownCloseHandlers();

    registerRuntimeEvents();

    window.addEventListener('DOMContentLoaded', async () => {
        await loadWorkspaces();
        await loadModels();
        await loadThreads(true);
        if (!state.currentThreadID) {
            await createNewThread();
        }
        dom.chatInput?.focus();
        setupViewSwitcher();
    });
}

function setupWorkspaceToggle() {
    if (!dom.workspaceToggle || !dom.workspaceSwitcher) return;
    dom.workspaceToggle.addEventListener('click', (e) => {
        e.stopPropagation();
        if (dom.workspaceToggle.disabled) return;
        const nextOpen = !dom.workspaceSwitcher.classList.contains('open');
        if (nextOpen) setModelDropdownOpen(false);
        setWorkspaceDropdownOpen(nextOpen);
    });
}

function setupModelToggle() {
    if (!dom.modelToggle || !dom.modelSwitcher) return;
    dom.modelToggle.addEventListener('click', (e) => {
        e.stopPropagation();
        if (dom.modelToggle.disabled) return;
        const nextOpen = !dom.modelSwitcher.classList.contains('open');
        if (nextOpen) setWorkspaceDropdownOpen(false);
        setModelDropdownOpen(nextOpen);
    });
}

function setupDropdownCloseHandlers() {
    document.addEventListener('click', (e) => {
        if (dom.workspaceSwitcher && !dom.workspaceSwitcher.contains(e.target)) setWorkspaceDropdownOpen(false);
        if (dom.modelSwitcher && !dom.modelSwitcher.contains(e.target)) setModelDropdownOpen(false);
    });
}

function registerRuntimeEvents() {
    if (!window.runtime?.EventsOn) return;
    window.runtime.EventsOn('agent:message', handleAgentMessage);
    window.runtime.EventsOn('agent:messages_truncated', handleMessagesTruncated);
    window.runtime.EventsOn('feed:item_pushed', handleFeedItemPushed);
    window.runtime.EventsOn('thread:title_updated', handleThreadTitleUpdated);
}
