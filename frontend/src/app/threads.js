import { dom } from './dom.js';
import { state } from './state.js';
import { getEventDurationSeconds, getEventText, safeParseJSON } from './utils.js';
import { appendTurnActions } from './chat/actions.js';
import { createTurnContainer, addUserMessage, appendResponseBlock, appendStatusMessage, appendThoughtBlock, appendToolCall, hideThinkingStatus, setChatEmptyState, showThinkingStatus, updateToolCall } from './chat/turns.js';
import { setTurnRegenerateContext } from './chat/regenerate.js';
import { setProcessingState } from './chat/processing.js';
import { syncCurrentThreadModel } from './models.js';
import { syncCurrentThreadWorkspace } from './workspaces.js';
import { isCancelMessage } from './chat/message-utils.js';
import { processAgentMessage } from './chat/agent.js';

export async function loadThreadMessages(threadID) {
    const shouldRestoreThinking = state.isProcessing;
    let restoredThinking = false;
    const loadToken = ++state.loadSequence;
    state.activeLoadToken = loadToken;
    state.activeLoadThreadID = threadID;
    state.isLoadingHistory = true;
    try {
        const turns = await window.go.app.App.GetThreadMessages(threadID);
        hideThinkingStatus();
        state.currentTurnContainer = null;
        state.currentToolCallElements.clear();
        state.lastThoughtText = '';
        state.lastThoughtElement = null;
        state.lastThinkingDuration = null;
        if (dom.chatMessages) {
            dom.chatMessages.innerHTML = '';
        }
        state.turnRegenerateContext.clear();
        state.userMessageCache.length = 0;
        state.userMessageCount = 0;

        state.lastAssistantTurnIndex = -1;
        for (let i = turns.length - 1; i >= 0; i -= 1) {
            if (turns[i]?.role === 'assistant') {
                state.lastAssistantTurnIndex = i;
                break;
            }
        }

        let userIndex = -1;
        for (let i = 0; i < turns.length; i += 1) {
            const turn = turns[i];
            if (!turn || !turn.events) continue;

            if (turn.role === 'user') {
                userIndex += 1;
                const userEvent = turn.events[0];
                if (userEvent && userEvent.type === 'user_message') {
                    const payload = safeParseJSON(userEvent.content);
                    if (payload && typeof payload === 'object' && payload.content) {
                        state.userMessageCache[userIndex] = payload.content;
                        addUserMessage(payload.content, false);
                    }
                }
                if (typeof state.userMessageCache[userIndex] !== 'string') {
                    state.userMessageCache[userIndex] = '';
                }
            } else if (turn.role === 'assistant') {
                createTurnContainer();
                if (userIndex >= 0) {
                    setTurnRegenerateContext(
                        state.currentTurnContainer,
                        userIndex,
                        state.userMessageCache[userIndex]
                    );
                }
                state.currentToolCallElements.clear();
                state.lastThoughtText = '';
                state.lastThoughtElement = null;
                state.lastThinkingDuration = null;
                let sawFinalResponse = false;
                let isCancelled = false;

                for (const event of turn.events) {
                    if (!event) continue;

                    switch (event.type) {
                        case 'start_thinking':
                            break;
                        case 'thought': {
                            const payload = safeParseJSON(event.content);
                            const duration = getEventDurationSeconds(payload);
                            if (payload && typeof payload === 'object') {
                                const reasoning = typeof payload.reasoning === 'string' ? payload.reasoning : '';
                                const content = typeof payload.content === 'string' ? payload.content : '';
                                if (reasoning.trim()) {
                                    appendThoughtBlock(reasoning, duration, true);
                                }
                                if (content.trim()) {
                                    appendResponseBlock(content);
                                }
                            } else {
                                const legacyText = getEventText(event.content);
                                if (legacyText && legacyText.trim()) {
                                    appendResponseBlock(legacyText);
                                }
                            }
                            state.lastThinkingDuration = typeof duration === 'number' ? duration : null;
                            break;
                        }
                        case 'executing_tool_start':
                            appendToolCall(safeParseJSON(event.content), true);
                            break;
                        case 'executing_tool_finish':
                            updateToolCall(safeParseJSON(event.content), true);
                            break;
                        case 'final_response': {
                            sawFinalResponse = true;
                            break;
                        }
                        case 'error': {
                            const historyErrorText = getEventText(event.content);
                            if (isCancelMessage(historyErrorText)) {
                                appendStatusMessage(state.currentTurnContainer, 'Cancelled', 'var(--text-tertiary)');
                                isCancelled = true;
                            } else {
                                appendStatusMessage(state.currentTurnContainer, '错误: ' + historyErrorText, 'var(--error-text)');
                            }
                            break;
                        }
                    }
                }
                const hasTurnContent = state.currentTurnContainer
                    ?.querySelector('.message-content')
                    ?.children.length > 0;
                if (hasTurnContent && !isCancelled) {
                    const allowRegenerate = sawFinalResponse;
                    appendTurnActions(state.currentTurnContainer, allowRegenerate);
                }
                state.currentTurnContainer = null;
                state.currentToolCallElements.clear();
            }
        }
        state.userMessageCount = state.userMessageCache.length;
        setChatEmptyState(!dom.chatMessages?.querySelector('.message-group'));
        if (shouldRestoreThinking && !state.currentThinkingBlock) {
            showThinkingStatus();
            restoredThinking = true;
        }
    } catch (error) {
        console.error('加载线程消息失败:', error);
    } finally {
        if (state.activeLoadToken !== loadToken) {
            return;
        }
        state.isLoadingHistory = false;
        if (state.pendingAgentMessages.length) {
            const queued = state.pendingAgentMessages
                .filter((item) => item.token === loadToken && item.threadID === threadID)
                .map((item) => item.data);
            state.pendingAgentMessages.length = 0;
            queued.forEach(processAgentMessage);
        }
        if (dom.chatMessages) {
            dom.chatMessages.scrollTop = dom.chatMessages.scrollHeight;
        }
        if (shouldRestoreThinking && !restoredThinking && !state.currentThinkingBlock) {
            showThinkingStatus();
            restoredThinking = true;
        }
        if (!restoredThinking) {
            state.currentTurnContainer = null;
        }
        state.currentToolCallElements.clear();
    }
}

export async function loadThreads(loadMessages = false) {
    try {
        state.threadsCache = await window.go.app.App.ListThreads();
        if (!state.currentThreadID && state.threadsCache.length > 0) {
            state.currentThreadID = state.threadsCache[0].id;
            loadMessages = true;
        }
        renderThreadList();
        syncCurrentThreadModel();
        syncCurrentThreadWorkspace();
        if (loadMessages && state.currentThreadID) {
            await loadThreadMessages(state.currentThreadID);
        }
    } catch (error) {
        console.error('加载线程列表失败:', error);
    }
}

export function renderThreadList() {
    if (!dom.threadList) return;
    dom.threadList.innerHTML = '';
    const term = state.searchTerm.toLowerCase();
    state.threadsCache
        .filter(thread => thread.title.toLowerCase().includes(term))
        .forEach(thread => dom.threadList.appendChild(createThreadItem(thread)));
}

function createThreadItem(thread) {
    const item = document.createElement('div');
    item.className = 'thread-item';
    if (thread.id === state.currentThreadID) item.classList.add('active');
    item.dataset.id = thread.id;
    item.draggable = true;

    item.innerHTML = `<div class="thread-text"></div><div class="thread-actions"><button class="thread-action-btn thread-delete-btn" title="删除">×</button></div>`;
    item.querySelector('.thread-text').textContent = thread.title;

    item.addEventListener('click', () => switchThread(thread.id));

    item.querySelector('.thread-delete-btn').addEventListener('click', async (e) => {
        e.stopPropagation();
        if (state.isProcessing) return;
        const wasCurrent = thread.id === state.currentThreadID;
        await window.go.app.App.DeleteThread(thread.id);
        if (wasCurrent) {
            state.currentThreadID = '';
            if (dom.chatMessages) {
                dom.chatMessages.innerHTML = '';
            }
        }
        await loadThreads(!wasCurrent);
    });

    item.addEventListener('dragstart', (e) => { state.dragSourceID = thread.id; e.dataTransfer.effectAllowed = 'move'; });
    item.addEventListener('dragover', (e) => { e.preventDefault(); item.classList.add('drag-over'); e.dataTransfer.dropEffect = 'move'; });
    item.addEventListener('dragleave', () => item.classList.remove('drag-over'));
    item.addEventListener('drop', async (e) => {
        e.preventDefault();
        item.classList.remove('drag-over');
        if (!state.dragSourceID || state.dragSourceID === thread.id) return;
        const fromIndex = state.threadsCache.findIndex(t => t.id === state.dragSourceID);
        const toIndex = state.threadsCache.findIndex(t => t.id === thread.id);
        const [moved] = state.threadsCache.splice(fromIndex, 1);
        state.threadsCache.splice(toIndex, 0, moved);
        renderThreadList();
        await persistThreadOrder();
    });

    return item;
}

export async function switchThread(threadID) {
    if (threadID === state.currentThreadID || state.isProcessing) return;

    state.currentThreadID = threadID;
    document.querySelectorAll('.thread-item.active').forEach(el => el.classList.remove('active'));
    document.querySelector(`.thread-item[data-id="${threadID}"]`)?.classList.add('active');

    syncCurrentThreadModel();
    syncCurrentThreadWorkspace();
    await loadThreadMessages(threadID);
    setProcessingState(false);
    dom.chatInput?.focus();
}

async function persistThreadOrder() {
    try {
        await window.go.app.App.ReorderThreads(state.threadsCache.map(t => t.id));
    } catch (error) {
        console.error('更新线程顺序失败:', error);
    }
}

export async function createNewThread() {
    if (state.isProcessing) return;
    try {
        state.currentThreadID = await window.go.app.App.CreateThread();
        if (dom.chatMessages) {
            dom.chatMessages.innerHTML = '';
        }
        setChatEmptyState(true);
        state.currentTurnContainer = null;
        state.currentToolCallElements.clear();
        state.lastThoughtText = '';
        state.lastThoughtElement = null;
        state.lastThinkingDuration = null;
        state.pendingRegenerateContext = null;
        state.turnRegenerateContext.clear();
        state.userMessageCache.length = 0;
        state.userMessageCount = 0;
        state.lastAssistantTurnIndex = -1;
        setProcessingState(false);
        await loadThreads(false);
        if (dom.threadList) {
            dom.threadList.scrollTop = 0;
        }
        dom.chatInput?.focus();
    } catch (error) {
        console.error('创建新线程失败:', error);
    }
}

export function setupThreadUI() {
    document.querySelector('#chat-view-sidebar .new-thread-btn')?.addEventListener('click', createNewThread);

    if (dom.searchBtn && dom.searchInput && dom.searchContainer) {
        const closeSearch = () => {
            dom.searchContainer.classList.remove('search-open');
            state.searchTerm = '';
            dom.searchInput.value = '';
            renderThreadList();
        };
        dom.searchBtn.addEventListener('click', () => {
            dom.searchContainer.classList.contains('search-open') ? closeSearch() : dom.searchContainer.classList.add('search-open');
            dom.searchInput.focus();
        });
        dom.searchInput.addEventListener('input', () => {
            state.searchTerm = dom.searchInput.value.trim();
            renderThreadList();
        });
        dom.searchInput.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                closeSearch();
                dom.searchInput.blur();
            }
        });
        dom.searchInput.addEventListener('blur', () => {
            if (!dom.searchInput.value.trim()) closeSearch();
        });
    }
}

export function handleThreadTitleUpdated(data) {
    const threadID = data?.thread_id || data?.threadID || data?.id;
    const title = data?.title;
    if (!threadID || typeof title !== 'string' || title.trim() === '') return;

    const thread = state.threadsCache.find(item => item.id === threadID);
    if (thread) {
        thread.title = title;
        renderThreadList();
        return;
    }

    loadThreads(false);
}

export function handleMessagesTruncated(data) {
    const threadID = data?.thread_id || data?.threadID;
    if (!threadID || threadID !== state.currentThreadID) return;
    loadThreadMessages(threadID);
}
