// 自动调整 textarea 高度
const chatInput = document.querySelector('.chat-input');

chatInput.addEventListener('input', function() {
    this.style.height = 'auto';
    this.style.height = Math.min(this.scrollHeight, 300) + 'px';
});

// DOM 元素
const sendBtn = document.querySelector('.send-btn');
const threadList = document.querySelector('.thread-list');
const chatMessages = document.querySelector('.chat-messages');
const searchInput = document.querySelector('.thread-search');
const searchBtn = document.querySelector('.search-btn');
const searchContainer = document.querySelector('#chat-view-sidebar .sidebar-actions');
const sidebar = document.querySelector('.sidebar');
const sidebarResizer = document.querySelector('.sidebar-resizer');
const workspaceSwitcher = document.querySelector('.workspace-switcher');
const workspaceToggle = document.querySelector('.workspace-toggle');
const workspaceToggleLabel = document.querySelector('.workspace-toggle-label');
const workspaceDropdown = document.querySelector('.workspace-dropdown');
const workspaceList = document.querySelector('.workspace-list');
const workspaceSelectBtn = document.querySelector('.workspace-select-btn');
const workspaceRefreshBtn = document.querySelector('.workspace-refresh');
const modelSwitcher = document.querySelector('.model-switcher');
const modelToggle = document.querySelector('.model-toggle');
const modelToggleLabel = document.querySelector('.model-toggle-label');
const modelDropdown = document.querySelector('.model-dropdown');
const modelList = document.querySelector('.model-list');
const modelSearchInput = document.querySelector('.model-search-input');
const viewSwitcher = document.querySelector('.view-switcher');

// 状态变量
let isProcessing = false;
// let currentAssistantMessage = null; // Replaced by currentTurnContainer
let currentTurnContainer = null;
let currentThinkingBlock = null;
let currentThreadID = '';
let threadsCache = [];
let searchTerm = '';
let dragSourceID = '';
let workspacesCache = [];
let currentWorkspacePath = '';
let modelsCache = [];
let modelSearchTerm = '';
let currentModelID = '';
let thinkingStartTime = 0;
let thinkingTimerInterval = null;
let currentToolCallElements = new Map();
let lastThoughtText = '';
let lastThoughtElement = null;
let lastThinkingDuration = null;
let lastAssistantTurnIndex = -1;

// 页面加载时初始化
window.addEventListener('DOMContentLoaded', async () => {
    await loadWorkspaces();
    await loadModels();
    await loadThreads(true); // 初始化时加载消息
    if (!currentThreadID) {
        await createNewThread();
    }
    chatInput.focus();
    setupViewSwitcher();
});

function setupViewSwitcher() {
    if (!viewSwitcher) return;
    viewSwitcher.addEventListener('click', (e) => {
        const viewBtn = e.target.closest('.view-btn');
        if (!viewBtn) return;

        const view = viewBtn.dataset.view;
        if (view) {
            switchView(view);
        }
    });
}

function switchView(viewName) {
    // Switch buttons
    document.querySelectorAll('.view-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.view === viewName);
    });

    // Switch sidebar content
    document.querySelectorAll('.sidebar-content .view-content').forEach(content => {
        content.classList.toggle('active', content.id === `${viewName}-view-sidebar`);
    });

    // Switch main content
    document.querySelectorAll('.main-content .view-content').forEach(content => {
        content.classList.toggle('active', content.id === `${viewName}-view-main`);
    });

    if (viewName === 'notifications') {
        loadFeedTopics();
    }
}

sendBtn.addEventListener('click', async () => {
    const message = chatInput.value.trim();
    if (message && !isProcessing) {
        if (!currentThreadID) {
            await createNewThread();
        }
        
        addUserMessage(message, true);
        chatInput.value = '';
        chatInput.style.height = 'auto';
        
        isProcessing = true;
        chatInput.disabled = true;
        sendBtn.disabled = true;
        updateWorkspaceToggleState();
        updateModelToggleState();
        lastThoughtText = '';
        lastThoughtElement = null;
        lastThinkingDuration = null;
        
        // The container will be created by the first agent event
        
        try {
            await window.go.app.App.AgentChat(currentThreadID, message);
        } catch (error) {
            console.error('Agent chat error:', error);
            handleError('抱歉，发生了错误: ' + error);
        }
    }
});

chatInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        sendBtn.click();
    }
});

function createTurnContainer() {
    if (currentTurnContainer) return;
    currentTurnContainer = document.createElement('div');
    currentTurnContainer.className = 'message-group assistant-message';
    const messageContent = document.createElement('div');
    messageContent.className = 'message-content';
    currentTurnContainer.appendChild(messageContent);
    chatMessages.appendChild(currentTurnContainer);
}

function showThinkingStatus() {
    createTurnContainer();
    
    // Don't show if there's already a thinking block
    if (currentThinkingBlock) return;

    currentThinkingBlock = document.createElement('div');
    currentThinkingBlock.className = 'thinking-status';
    currentThinkingBlock.innerHTML = `
        <span class="tool-icon">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <path d="m12 3-1.9 4.8-4.8 1.9 4.8 1.9 1.9 4.8 1.9-4.8 4.8-1.9-4.8-1.9L12 3z"/>
                <path d="M5 3v4"/>
                <path d="M19 17v4"/>
                <path d="M3 5h4"/>
                <path d="M17 19h4"/>
            </svg>
        </span>
        <span class="thinking-text-label">Thinking</span>
        <span class="thinking-dots">
            <span class="dot">.</span>
            <span class="dot">.</span>
            <span class="dot">.</span>
        </span>
        <span class="thinking-timer">0.0s</span>
    `;
    currentThinkingBlock.style.display = 'flex';
    
    currentTurnContainer.querySelector('.message-content').appendChild(currentThinkingBlock);
    
    const timerElement = currentThinkingBlock.querySelector('.thinking-timer');
    thinkingStartTime = Date.now();
    thinkingTimerInterval = setInterval(() => {
        const elapsed = ((Date.now() - thinkingStartTime) / 1000).toFixed(1);
        timerElement.textContent = `${elapsed}s`;
    }, 100);
}

function hideThinkingStatus() {
    if (thinkingTimerInterval) {
        clearInterval(thinkingTimerInterval);
        thinkingTimerInterval = null;
    }
    if (currentThinkingBlock) {
        currentThinkingBlock.remove();
        currentThinkingBlock = null;
    }
}

function appendThoughtBlock(text, durationInSeconds) {
    createTurnContainer();

    const thoughtBlock = document.createElement('div');
    thoughtBlock.className = 'thought-block';

    const durationText = durationInSeconds ? ` for ${durationInSeconds.toFixed(2)}s` : '';

    thoughtBlock.innerHTML = `
        <div class="thought-header">
            <span class="tool-icon">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                    <path d="m12 3-1.9 4.8-4.8 1.9 4.8 1.9 1.9 4.8 1.9-4.8 4.8-1.9-4.8-1.9L12 3z"/>
                    <path d="M5 3v4"/>
                    <path d="M19 17v4"/>
                    <path d="M3 5h4"/>
                    <path d="M17 19h4"/>
                </svg>
            </span>
            <span>Thought${durationText}</span>
        </div>
        <div class="thought-content">
            <div class="markdown-body"></div>
        </div>
    `;
    
    let displayText = text;
    if (typeof displayText !== 'string') {
        if (displayText && typeof displayText.content === 'string') {
            displayText = displayText.content;
        } else {
            displayText = JSON.stringify(displayText, null, 2);
        }
    }
    renderMarkdownInto(thoughtBlock.querySelector('.markdown-body'), displayText);
    lastThoughtText = typeof displayText === 'string' ? displayText.trim() : String(displayText).trim();
    lastThoughtElement = thoughtBlock;

    currentTurnContainer.querySelector('.message-content').appendChild(thoughtBlock);
    chatMessages.scrollTop = chatMessages.scrollHeight;
}

// 处理来自 Go 后端的 agent 消息
function handleAgentMessage(data) {
    const { type, content } = data;
    
    switch (type) {
        case 'start_thinking':
            showThinkingStatus();
            break;
        case 'thought':
            const duration = thinkingStartTime ? (Date.now() - thinkingStartTime) / 1000 : null;
            hideThinkingStatus();
            appendThoughtBlock(getEventText(content), duration);
            lastThinkingDuration = duration;
            break;
        case 'executing_tool_start':
            hideThinkingStatus();
            appendToolCall(safeParseJSON(content));
            break;
        case 'executing_tool_finish':
            updateToolCall(safeParseJSON(content));
            break;
        case 'final_response':
            handleFinalResponse();
            break;
        case 'error':
            handleError(getEventText(content));
            break;
    }
}

function handleFinalResponse() {
    hideThinkingStatus();
    if (currentTurnContainer) {
        appendTurnActions(currentTurnContainer, true);
    }
    isProcessing = false;
    chatInput.disabled = false;
    sendBtn.disabled = false;
    updateWorkspaceToggleState();
    updateModelToggleState();
    chatInput.focus();
    currentTurnContainer = null;
    currentToolCallElements.clear();
    lastThoughtText = '';
    lastThoughtElement = null;
    lastThinkingDuration = null;
    loadThreads(false); // 刷新线程列表，但不重新加载消息
}

function handleError(errorMessage) {
    hideThinkingStatus();
    if (!currentTurnContainer) {
        createTurnContainer();
    }
    const p = document.createElement('p');
    p.style.color = 'var(--error-text)';
    p.textContent = '错误: ' + errorMessage;
    currentTurnContainer.querySelector('.message-content').appendChild(p);
    
    isProcessing = false;
    chatInput.disabled = false;
    sendBtn.disabled = false;
    updateWorkspaceToggleState();
    updateModelToggleState();
    chatInput.focus();
    currentTurnContainer = null;
    currentToolCallElements.clear();
    lastThoughtText = '';
    lastThoughtElement = null;
    lastThinkingDuration = null;
}

window.runtime.EventsOn('agent:message', handleAgentMessage);
window.runtime.EventsOn('agent:messages_truncated', handleMessagesTruncated);
window.runtime.EventsOn('feed:item_pushed', handleFeedItemPushed);

function addUserMessage(text, scrollToBottom = true) {
    const messageGroup = document.createElement('div');
    messageGroup.className = 'message-group user-message';
    
    const messageContent = document.createElement('div');
    messageContent.className = 'message-content';
    
    const p = document.createElement('p');
    
    let displayText = text;
    if (typeof displayText !== 'string') {
        if (displayText && typeof displayText.content === 'string') {
            displayText = displayText.content;
        } else {
            displayText = JSON.stringify(displayText, null, 2);
        }
    }
    p.textContent = displayText;
    
    messageContent.appendChild(p);
    messageGroup.appendChild(messageContent);
    
    chatMessages.appendChild(messageGroup);
    if (scrollToBottom) {
        chatMessages.scrollTop = chatMessages.scrollHeight;
    }
}

async function loadThreadMessages(threadID) {
    try {
        const turns = await window.go.app.App.GetThreadMessages(threadID);
        chatMessages.innerHTML = '';

        lastAssistantTurnIndex = -1;
        for (let i = turns.length - 1; i >= 0; i -= 1) {
            if (turns[i]?.role === 'assistant') {
                lastAssistantTurnIndex = i;
                break;
            }
        }

        for (let i = 0; i < turns.length; i += 1) {
            const turn = turns[i];
            if (!turn || !turn.events) continue;

            if (turn.role === 'user') {
                const userEvent = turn.events[0];
                if (userEvent && userEvent.type === 'user_message') {
                    const payload = safeParseJSON(userEvent.content);
                    if (payload && typeof payload === 'object' && payload.content) {
                        addUserMessage(payload.content, false);
                    }
                }
            } else if (turn.role === 'assistant') {
                createTurnContainer(); // Create the container for the turn
                currentToolCallElements.clear();
                lastThoughtText = '';
                lastThoughtElement = null;
                lastThinkingDuration = null;
                let sawFinalResponse = false;

                for (const event of turn.events) {
                    if (!event) continue;
                    
                    switch (event.type) {
                        case 'start_thinking':
                            // Don't show live thinking animation for history
                            break;
                        case 'thought': {
                            const payload = safeParseJSON(event.content);
                            const duration = getEventDurationSeconds(payload);
                            appendThoughtBlock(getEventText(event.content), duration);
                            lastThinkingDuration = typeof duration === 'number' ? duration : null;
                            break;
                        }
                        case 'executing_tool_start':
                            appendToolCall(safeParseJSON(event.content));
                            break;
                        case 'executing_tool_finish':
                            updateToolCall(safeParseJSON(event.content), true);
                            break;
                        case 'final_response': {
                            sawFinalResponse = true;
                            break;
                        }
                        case 'error':
                             const p = document.createElement('p');
                             p.style.color = 'var(--error-text)';
                             p.textContent = '错误: ' + getEventText(event.content);
                             currentTurnContainer.querySelector('.message-content').appendChild(p);
                            break;
                    }
                }
                if (sawFinalResponse) {
                    const allowRegenerate = i === lastAssistantTurnIndex;
                    appendTurnActions(currentTurnContainer, allowRegenerate);
                }
                currentTurnContainer = null;
                currentToolCallElements.clear();
            }
        }
    } catch (error) {
        console.error('加载线程消息失败:', error);
    } finally {
        chatMessages.scrollTop = chatMessages.scrollHeight;
        // Ensure state is clean
        currentTurnContainer = null;
        currentToolCallElements.clear();
    }
}

function appendToolCall(payload) {
    if (!payload || typeof payload !== 'object' || !payload.id) return;
    createTurnContainer();
    
    const toolCallsContainer = currentTurnContainer.querySelector('.message-content');
    if (!toolCallsContainer) return;

    const toolCallEl = document.createElement('div');
    toolCallEl.className = 'tool-call collapsed';
    toolCallEl.dataset.toolCallId = payload.id;
    toolCallEl.dataset.startTime = Date.now();

    toolCallEl.innerHTML = `
        <div class="tool-call-header">
            <span class="tool-icon">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                    <path d="M14.7 6.3a4 4 0 0 0-5.7 5.6L4 16.9 7.1 20l5-5a4 4 0 0 0 5.6-5.7l-2.2 2.2-2.2-2.2 2.4-3.0z"/>
                </svg>
            </span>
            <span class="tool-name">${payload.name}</span>
            <span class="duration"></span>
            <span class="status-badge running">Running</span>
            <button class="expand-btn">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                    <polyline points="6 9 12 15 18 9"></polyline>
                </svg>
            </button>
        </div>
        <div class="tool-call-body">
            <div class="tool-call-section">
                <div class="tool-call-section-title">ARGUMENTS</div>
                <div class="tool-call-content"><pre></pre></div>
            </div>
            <div class="tool-call-section result-section" style="display: none;">
                <div class="tool-call-section-title">RESULT</div>
                <div class="tool-call-content result-content"><pre></pre></div>
            </div>
        </div>
    `;

    toolCallEl.querySelector('.tool-call-content pre').textContent = formatToolArgs(payload.args);
    
    toolCallEl.querySelector('.tool-call-header').addEventListener('click', () => {
        toolCallEl.classList.toggle('collapsed');
    });

    toolCallsContainer.appendChild(toolCallEl);
    currentToolCallElements.set(payload.id, toolCallEl);
    chatMessages.scrollTop = chatMessages.scrollHeight;
}

function updateToolCall(payload, isHistory = false) {
    if (!payload || typeof payload !== 'object' || !payload.id || !currentToolCallElements.has(payload.id)) return;

    const toolCallEl = currentToolCallElements.get(payload.id);
    
    const statusBadge = toolCallEl.querySelector('.status-badge');
    statusBadge.textContent = 'Completed';
    statusBadge.classList.remove('running');
    statusBadge.classList.add('completed');

    const durationEl = toolCallEl.querySelector('.duration');
    if (isHistory) {
        const durationSeconds = getEventDurationSeconds(payload);
        if (typeof durationSeconds === 'number') {
            durationEl.textContent = `${durationSeconds.toFixed(2)}s`;
        }
    } else {
        const startTime = parseInt(toolCallEl.dataset.startTime, 10);
        const duration = ((Date.now() - startTime) / 1000);
        durationEl.textContent = `${duration.toFixed(2)}s`;
    }

    const resultSection = toolCallEl.querySelector('.result-section');
    resultSection.style.display = 'block';
    
    const resultContent = toolCallEl.querySelector('.result-content pre');
    let textContent = payload.content;
    if (typeof textContent !== 'string') {
        textContent = JSON.stringify(textContent, null, 2);
    }
    resultContent.textContent = textContent;

    // Do not delete from map if we are in history mode and might have more updates
    if (!isHistory) {
        currentToolCallElements.delete(payload.id);
    }
    chatMessages.scrollTop = chatMessages.scrollHeight;
}


// --- Rest of the file (Workspace, Model, and Thread logic) remains largely the same ---

async function loadWorkspaces() {
    if (!window.go?.app?.App?.ListWorkspaces) return;
    try {
        workspacesCache = await window.go.app.App.ListWorkspaces();
    } catch (error) {
        console.error('加载工作目录失败:', error);
        workspacesCache = [];
    }
    syncCurrentThreadWorkspace();
}

function updateWorkspaceToggle() {
    if (!workspaceToggleLabel) return;
    const active = workspacesCache.find(workspace => workspace.path === currentWorkspacePath);
    workspaceToggleLabel.textContent = active ? getWorkspaceName(active.path) : (currentWorkspacePath ? getWorkspaceName(currentWorkspacePath) : 'Workspace');
}

function updateWorkspaceToggleState() {
    if (!workspaceToggle) return;
    workspaceToggle.disabled = isProcessing || !currentThreadID || workspacesCache.length === 0;
}

function setWorkspaceDropdownOpen(isOpen) {
    if (!workspaceSwitcher || !workspaceDropdown || !workspaceToggle) return;
    workspaceSwitcher.classList.toggle('open', isOpen);
    workspaceToggle.setAttribute('aria-expanded', String(isOpen));
    workspaceDropdown.setAttribute('aria-hidden', String(!isOpen));
}

function renderWorkspaceList() {
    if (!workspaceList) return;
    workspaceList.innerHTML = '';
    if (workspacesCache.length === 0) {
        workspaceList.innerHTML = '<div class="workspace-empty">No workspace available</div>';
        return;
    }

    workspacesCache.forEach(workspace => {
        const item = document.createElement('button');
        item.type = 'button';
        item.className = 'workspace-item';
        if (workspace.path === currentWorkspacePath) item.classList.add('active');
        item.innerHTML = `
            <span class="workspace-item-icon">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M3 7a2 2 0 0 1 2-2h3l2 2h9a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"></path>
                </svg>
            </span>
            <div class="workspace-item-content">
                <span class="workspace-item-name">${getWorkspaceName(workspace.path)}</span>
                <span class="workspace-item-path">${workspace.path}</span>
            </div>
        `;
        item.addEventListener('click', () => setWorkspace(workspace.path));
        workspaceList.appendChild(item);
    });
}

async function setWorkspace(workspacePath) {
    if (!workspacePath || !currentThreadID || isProcessing || workspacePath === currentWorkspacePath) {
        setWorkspaceDropdownOpen(false);
        return;
    }
    try {
        await window.go.app.App.UpdateWorkspace(currentThreadID, workspacePath);
        currentWorkspacePath = workspacePath;
        const thread = threadsCache.find(item => item.id === currentThreadID);
        if (thread) thread.work_dir = workspacePath;
        await loadWorkspaces();
        setWorkspaceDropdownOpen(false);
    } catch (error) {
        console.error('切换工作目录失败:', error);
    }
}

async function selectWorkspaceDirectory() {
    if (isProcessing || !currentThreadID) return;
    try {
        const selectedPath = await window.go.app.App.SelectWorkspace(currentThreadID);
        if (selectedPath) {
            currentWorkspacePath = selectedPath;
            const thread = threadsCache.find(item => item.id === currentThreadID);
            if (thread) thread.work_dir = selectedPath;
        }
        await loadWorkspaces();
    } catch (error) {
        console.error('选择工作目录失败:', error);
    }
}

async function resetWorkspace() {
    if (isProcessing || !currentThreadID) return;
    const fallback = getDefaultWorkspacePath();
    if (fallback) await setWorkspace(fallback);
}

async function loadModels() {
    if (!window.go?.app?.App?.ListModels) return;
    try {
        modelsCache = await window.go.app.App.ListModels();
    } catch (error) {
        console.error('加载模型列表失败:', error);
        modelsCache = [];
    }
    renderModelList();
    updateModelToggle();
    updateModelToggleState();
}

function updateModelToggle() {
    if (!modelToggleLabel) return;
    const currentModel = modelsCache.find(model => model.id === currentModelID);
    modelToggleLabel.textContent = currentModel ? currentModel.name : 'Select model';
}

function updateModelToggleState() {
    if (!modelToggle) return;
    modelToggle.disabled = !currentThreadID || isProcessing || modelsCache.length === 0;
}

function syncCurrentThreadModel() {
    const thread = threadsCache.find(item => item.id === currentThreadID);
    currentModelID = thread ? thread.model : '';
    updateModelToggle();
    renderModelList();
    updateModelToggleState();
}

function syncCurrentThreadWorkspace() {
    const thread = threadsCache.find(item => item.id === currentThreadID);
    currentWorkspacePath = thread?.work_dir || getDefaultWorkspacePath() || '';
    updateWorkspaceToggle();
    renderWorkspaceList();
    updateWorkspaceToggleState();
}

function setModelDropdownOpen(isOpen) {
    if (!modelSwitcher || !modelDropdown || !modelToggle) return;
    modelSwitcher.classList.toggle('open', isOpen);
    modelToggle.setAttribute('aria-expanded', String(isOpen));
    modelDropdown.setAttribute('aria-hidden', String(!isOpen));
    if (isOpen && modelSearchInput) modelSearchInput.focus();
}

function renderModelList() {
    if (!modelList) return;
    modelList.innerHTML = '';
    const term = modelSearchTerm.toLowerCase();
    const filtered = modelsCache.filter(m => !term || m.name.toLowerCase().includes(term) || m.id.toLowerCase().includes(term) || (m.provider || '').toLowerCase().includes(term));

    if (filtered.length === 0) {
        modelList.innerHTML = `<div class="model-empty">${modelsCache.length === 0 ? 'No models available' : 'No models found'}</div>`;
        return;
    }

    const groups = filtered.reduce((acc, model) => {
        const provider = model.provider || 'Models';
        if (!acc[provider]) acc[provider] = [];
        acc[provider].push(model);
        return acc;
    }, {});

    Object.entries(groups).forEach(([provider, models]) => {
        const groupEl = document.createElement('div');
        groupEl.className = 'model-group';
        groupEl.innerHTML = `<div class="model-group-title">${provider}</div>`;
        models.forEach(model => {
            const option = document.createElement('button');
            option.type = 'button';
            option.className = 'model-option';
            if (model.id === currentModelID) option.classList.add('selected');
            option.innerHTML = `<span class="model-option-name">${model.name}</span><span class="model-option-meta">${model.context_window}</span>`;
            option.addEventListener('click', () => selectModel(model.id));
            groupEl.appendChild(option);
        });
        modelList.appendChild(groupEl);
    });
}

async function selectModel(modelID) {
    if (!currentThreadID || isProcessing || modelID === currentModelID) {
        setModelDropdownOpen(false);
        return;
    }
    try {
        await window.go.app.App.UpdateThreadModel(currentThreadID, modelID);
        currentModelID = modelID;
        const thread = threadsCache.find(item => item.id === currentThreadID);
        if (thread) thread.model = modelID;
        updateModelToggle();
        renderModelList();
        setModelDropdownOpen(false);
    } catch (error) {
        console.error('切换模型失败:', error);
    }
}

async function loadThreads(loadMessages = false) {
    try {
        threadsCache = await window.go.app.App.ListThreads();
        if (!currentThreadID && threadsCache.length > 0) {
            currentThreadID = threadsCache[0].id;
            loadMessages = true;
        }
        renderThreadList();
        syncCurrentThreadModel();
        syncCurrentThreadWorkspace();
        if (loadMessages && currentThreadID) {
            await loadThreadMessages(currentThreadID);
        }
    } catch (error) {
        console.error('加载线程列表失败:', error);
    }
}

function renderThreadList() {
    threadList.innerHTML = '';
    const term = searchTerm.toLowerCase();
    threadsCache
        .filter(thread => thread.title.toLowerCase().includes(term))
        .forEach(thread => threadList.appendChild(createThreadItem(thread)));
}

function createThreadItem(thread) {
    const item = document.createElement('div');
    item.className = 'thread-item';
    if (thread.id === currentThreadID) item.classList.add('active');
    item.dataset.id = thread.id;
    item.draggable = true;
    
    item.innerHTML = `<div class="thread-text"></div><div class="thread-actions"><button class="thread-action-btn thread-delete-btn" title="删除">×</button></div>`;
    item.querySelector('.thread-text').textContent = thread.title;

    item.addEventListener('click', () => switchThread(thread.id));

    item.querySelector('.thread-delete-btn').addEventListener('click', async (e) => {
        e.stopPropagation();
        if (isProcessing) return;
        const wasCurrent = thread.id === currentThreadID;
        await window.go.app.App.DeleteThread(thread.id);
        if (wasCurrent) {
            currentThreadID = '';
            chatMessages.innerHTML = '';
        }
        await loadThreads(!wasCurrent);
    });

    item.addEventListener('dragstart', (e) => { dragSourceID = thread.id; e.dataTransfer.effectAllowed = 'move'; });
    item.addEventListener('dragover', (e) => { e.preventDefault(); item.classList.add('drag-over'); e.dataTransfer.dropEffect = 'move'; });
    item.addEventListener('dragleave', () => item.classList.remove('drag-over'));
    item.addEventListener('drop', async (e) => {
        e.preventDefault();
        item.classList.remove('drag-over');
        if (!dragSourceID || dragSourceID === thread.id) return;
        const fromIndex = threadsCache.findIndex(t => t.id === dragSourceID);
        const toIndex = threadsCache.findIndex(t => t.id === thread.id);
        const [moved] = threadsCache.splice(fromIndex, 1);
        threadsCache.splice(toIndex, 0, moved);
        renderThreadList();
        await persistThreadOrder();
    });

    return item;
}

async function switchThread(threadID) {
    if (threadID === currentThreadID || isProcessing) return;
    
    currentThreadID = threadID;
    document.querySelectorAll('.thread-item.active').forEach(el => el.classList.remove('active'));
    document.querySelector(`.thread-item[data-id="${threadID}"]`)?.classList.add('active');

    syncCurrentThreadModel();
    syncCurrentThreadWorkspace();
    await loadThreadMessages(threadID);
}

async function persistThreadOrder() {
    try {
        await window.go.app.App.ReorderThreads(threadsCache.map(t => t.id));
    } catch (error) {
        console.error('更新线程顺序失败:', error);
    }
}



async function createNewThread() {
    if (isProcessing) return;
    try {
        currentThreadID = await window.go.app.App.CreateThread();
        chatMessages.innerHTML = '';
        await loadThreads(false);
    } catch (error) {
        console.error('创建新线程失败:', error);
    }
}

// Event Listeners Setup
document.querySelector('#chat-view-sidebar .new-thread-btn')?.addEventListener('click', createNewThread);

if (workspaceToggle) {
    workspaceToggle.addEventListener('click', (e) => {
        e.stopPropagation();
        if (workspaceToggle.disabled) return;
        setWorkspaceDropdownOpen(!workspaceSwitcher.classList.contains('open'));
    });
}
workspaceSelectBtn?.addEventListener('click', async (e) => { e.stopPropagation(); await selectWorkspaceDirectory(); setWorkspaceDropdownOpen(false); });
workspaceRefreshBtn?.addEventListener('click', async (e) => { e.stopPropagation(); await resetWorkspace(); });

if (modelToggle) {
    modelToggle.addEventListener('click', (e) => {
        e.stopPropagation();
        if (modelToggle.disabled) return;
        setModelDropdownOpen(!modelSwitcher.classList.contains('open'));
    });
}
if (modelSearchInput) {
    modelSearchInput.addEventListener('input', () => { modelSearchTerm = modelSearchInput.value.trim().toLowerCase(); renderModelList(); });
    modelSearchInput.addEventListener('keydown', (e) => { if (e.key === 'Escape') { setModelDropdownOpen(false); modelToggle?.focus(); } });
}

document.addEventListener('click', (e) => {
    if (workspaceSwitcher && !workspaceSwitcher.contains(e.target)) setWorkspaceDropdownOpen(false);
    if (modelSwitcher && !modelSwitcher.contains(e.target)) setModelDropdownOpen(false);
});

if (searchBtn && searchInput && searchContainer) {
    const closeSearch = () => { searchContainer.classList.remove('search-open'); searchTerm = ''; searchInput.value = ''; renderThreadList(); };
    searchBtn.addEventListener('click', () => { searchContainer.classList.contains('search-open') ? closeSearch() : searchContainer.classList.add('search-open'); searchInput.focus(); });
    searchInput.addEventListener('input', () => { searchTerm = searchInput.value.trim(); renderThreadList(); });
    searchInput.addEventListener('keydown', (e) => { if (e.key === 'Escape') { closeSearch(); searchInput.blur(); } });
    searchInput.addEventListener('blur', () => { if (!searchInput.value.trim()) closeSearch(); });
}

if (sidebarResizer) {
    sidebarResizer.addEventListener('mousedown', (e) => {
        e.preventDefault();
        const startX = e.clientX;
        const startWidth = sidebar.getBoundingClientRect().width;
        const onMouseMove = (moveEvent) => {
            const nextWidth = Math.min(360, Math.max(180, startWidth + (moveEvent.clientX - startX)));
            sidebar.style.width = `${Math.round(nextWidth)}px`;
        };
        const onMouseUp = () => {
            document.removeEventListener('mousemove', onMouseMove);
            document.removeEventListener('mouseup', onMouseUp);
            document.body.style.cursor = '';
            document.body.style.userSelect = '';
        };
        document.addEventListener('mousemove', onMouseMove);
        document.addEventListener('mouseup', onMouseUp);
        document.body.style.cursor = 'col-resize';
        document.body.style.userSelect = 'none';
    });
}

// Utility Functions
function safeParseJSON(text) {
    try {
        const normalized = normalizeEventContent(text);
        if (normalized && typeof normalized === 'object') {
            return normalized;
        }
        return JSON.parse(normalized);
    } catch (error) {
        return null;
    }
}

function formatToolArgs(args) {
    if (args == null) return '';
    if (typeof args === 'string') {
        try {
            const parsed = JSON.parse(args);
            if (parsed && typeof parsed === 'object') {
                return JSON.stringify(parsed, null, 4);
            }
        } catch (error) {
            return args;
        }
        return args;
    }
    return JSON.stringify(args, null, 4);
}

function getEventText(content) {
    const payload = safeParseJSON(content);
    if (payload && typeof payload === 'object') {
        if (typeof payload.content === 'string') return payload.content;
        if (typeof payload.error === 'string') return payload.error;
    }
    return typeof content === 'string' ? content : '';
}

function appendTurnActions(container, allowRegenerate) {
    if (!container) return;
    if (container.querySelector('.message-actions')) return;

    const actions = document.createElement('div');
    actions.className = 'message-actions';

    const copyBtn = document.createElement('button');
    copyBtn.className = 'icon-btn';
    copyBtn.title = '复制';
    copyBtn.innerHTML = `
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <rect x="9" y="9" width="13" height="13" rx="2"></rect>
            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
        </svg>
    `;
    copyBtn.addEventListener('click', () => handleCopyTurn(container));

    const regenBtn = document.createElement('button');
    regenBtn.className = 'icon-btn';
    regenBtn.title = allowRegenerate ? '重新生成' : '仅支持最新回复重新生成';
    regenBtn.innerHTML = `
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M3 12a9 9 0 0 1 14-7l2 2"></path>
            <path d="M21 12a9 9 0 0 1-14 7l-2-2"></path>
            <path d="M17 3h4v4"></path>
            <path d="M7 21H3v-4"></path>
        </svg>
    `;
    if (!allowRegenerate) {
        regenBtn.classList.add('is-disabled');
    } else {
        regenBtn.addEventListener('click', handleRegenerateTurn);
    }

    actions.appendChild(copyBtn);
    actions.appendChild(regenBtn);
    container.querySelector('.message-content').appendChild(actions);
}

function getTurnCopyText(container) {
    if (!container) return '';
    const parts = [];
    const blocks = container.querySelectorAll('.thought-content, .message-content > p');
    blocks.forEach((block) => {
        const text = block.textContent?.trim();
        if (text) parts.push(text);
    });
    return parts.join('\n\n');
}

async function handleCopyTurn(container) {
    const text = getTurnCopyText(container);
    if (!text) return;
    try {
        await navigator.clipboard.writeText(text);
    } catch (error) {
        const textarea = document.createElement('textarea');
        textarea.value = text;
        textarea.style.position = 'fixed';
        textarea.style.left = '-9999px';
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand('copy');
        textarea.remove();
    }
}

async function handleRegenerateTurn() {
    if (!currentThreadID || isProcessing) return;
    isProcessing = true;
    chatInput.disabled = true;
    sendBtn.disabled = true;
    updateWorkspaceToggleState();
    updateModelToggleState();
    lastThoughtText = '';
    lastThoughtElement = null;
    lastThinkingDuration = null;

    try {
        await window.go.app.App.RegenerateLastResponse(currentThreadID);
    } catch (error) {
        console.error('Regenerate response error:', error);
        handleError('抱歉，发生了错误: ' + error);
    }
}

function handleMessagesTruncated(data) {
    const threadID = data?.thread_id || data?.threadID;
    if (!threadID || threadID !== currentThreadID) return;
    loadThreadMessages(threadID);
}

function getEventDurationSeconds(payload) {
    if (!payload || typeof payload !== 'object') return null;
    const duration = payload.duration_seconds;
    if (typeof duration === 'number' && Number.isFinite(duration)) {
        return duration;
    }
    return null;
}

function normalizeEventContent(content) {
    if (content == null || typeof content === 'string') {
        return content;
    }
    if (content instanceof Uint8Array) {
        return new TextDecoder().decode(content);
    }
    if (Array.isArray(content)) {
        return new TextDecoder().decode(new Uint8Array(content));
    }
    return content;
}

function getWorkspaceName(path) {
    return path?.split('/').filter(Boolean).pop() || path || '';
}

function getDefaultWorkspacePath() {
    const fallback = workspacesCache.find(w => w.is_default);
    return fallback ? fallback.path : (workspacesCache[0]?.path || '');
}

// --- Feed View ---
const feedTopicsList = document.querySelector('.feed-topics-list');
const feedArticlesList = document.querySelector('.feed-articles-list');
const feedDetailView = document.querySelector('#feed-detail-view');
const feedBackBtn = document.querySelector('#feed-back-btn');
const feedDetailTitle = document.querySelector('#feed-detail-title');
const feedDetailTimestamp = document.querySelector('#feed-detail-timestamp');
const feedDetailContent = document.querySelector('#feed-detail-content');

let feedTopicsCache = [];
let feedArticlesCache = [];
let currentFeedTopicId = '';
let feedTopicsLoading = false;
let feedArticlesLoading = false;
let feedTopicsError = '';
let feedArticlesError = '';
let feedArticlesRequestToken = 0;

if (window.marked?.setOptions) {
    window.marked.setOptions({
        gfm: true,
        breaks: true
    });
}

if (feedBackBtn) {
    feedBackBtn.addEventListener('click', () => {
        if (feedDetailView) feedDetailView.style.display = 'none';
        if (feedArticlesList) feedArticlesList.style.display = 'block';
    });
}
function handleFeedItemPushed(item) {
    const topic = item?.topic;
    if (!topic) return;
    if (!feedTopicsCache.includes(topic)) {
        feedTopicsCache = [...feedTopicsCache, topic];
    }
    if (!currentFeedTopicId) {
        currentFeedTopicId = topic;
    }
    renderFeedTopics();
    if (topic !== currentFeedTopicId) return;
    let updated = false;
    if (item?.id) {
        const existingIndex = feedArticlesCache.findIndex(article => article?.id === item.id);
        if (existingIndex !== -1) {
            feedArticlesCache[existingIndex] = item;
            updated = true;
        }
    }
    if (!updated) {
        feedArticlesCache = [item, ...feedArticlesCache];
    }
    feedArticlesCache.sort((a, b) => (b?.created_at || 0) - (a?.created_at || 0));
    renderFeedArticles();
}
function renderFeedTopics() {
    if (!feedTopicsList) return;
    feedTopicsList.innerHTML = '';
    if (feedTopicsLoading) {
        feedTopicsList.innerHTML = '<div class="feed-empty">正在加载主题...</div>';
        return;
    }
    if (feedTopicsError) {
        feedTopicsList.innerHTML = `<div class="feed-empty">${feedTopicsError}</div>`;
        return;
    }
    if (feedTopicsCache.length === 0) {
        feedTopicsList.innerHTML = '<div class="feed-empty">暂无主题</div>';
        return;
    }
    feedTopicsCache.forEach(topic => {
        const topicEl = document.createElement('div');
        topicEl.className = 'feed-topic-item';
        topicEl.innerHTML = `<div class="thread-text"></div>`;
        topicEl.querySelector('.thread-text').textContent = topic;
        topicEl.dataset.topicId = topic;
        if (topic === currentFeedTopicId) {
            topicEl.classList.add('active');
        }
        topicEl.addEventListener('click', async () => {
            // Return to the list view if the detail view is open
            if (feedDetailView) feedDetailView.style.display = 'none';
            if (feedArticlesList) feedArticlesList.style.display = 'block';

            if (topic === currentFeedTopicId) {
                // Same topic, no need to reload
                return;
            }

            currentFeedTopicId = topic;
            renderFeedTopics();
            await loadFeedArticles(topic);
        });
        feedTopicsList.appendChild(topicEl);
    });
}
function renderFeedArticles() {
    if (!feedArticlesList) return;
    if (feedArticlesLoading) {
        feedArticlesList.innerHTML = '<div class="feed-empty">正在加载内容...</div>';
        return;
    }
    if (!currentFeedTopicId) {
        feedArticlesList.innerHTML = '<div class="feed-empty">选择一个主题以查看内容。</div>';
        return;
    }
    if (feedArticlesError) {
        feedArticlesList.innerHTML = `<div class="feed-empty">${feedArticlesError}</div>`;
        return;
    }
    if (feedArticlesCache.length === 0) {
        feedArticlesList.innerHTML = '<div class="feed-empty">暂无内容</div>';
        return;
    }
    feedArticlesList.innerHTML = '';
    feedArticlesCache.forEach(article => {
        const articleEl = document.createElement('div');
        articleEl.className = 'feed-article-item';
        
        const headerEl = document.createElement('div');
        headerEl.className = 'feed-article-header';
        const titleEl = document.createElement('h2');
        titleEl.className = 'feed-article-title';
        titleEl.textContent = article?.title || '未命名';
        const timestampEl = document.createElement('span');
        timestampEl.className = 'feed-article-timestamp';
        timestampEl.textContent = formatFeedTimestamp(article?.created_at);
        const contentEl = document.createElement('p');
        contentEl.className = 'feed-article-content';
        contentEl.textContent = article?.content || '';

        headerEl.appendChild(titleEl);
        headerEl.appendChild(timestampEl);
        articleEl.appendChild(headerEl);
        articleEl.appendChild(contentEl);

        articleEl.addEventListener('click', () => {
            if (feedDetailTitle) feedDetailTitle.textContent = article?.title || '未命名';
            if (feedDetailTimestamp) feedDetailTimestamp.textContent = formatFeedTimestamp(article?.created_at);
            renderFeedDetailMarkdown(article?.content || '');
            if (feedArticlesList) feedArticlesList.style.display = 'none';
            if (feedDetailView) feedDetailView.style.display = 'flex';
        });
        feedArticlesList.appendChild(articleEl);
    });
}

async function loadFeedTopics() {
    if (!window.go?.app?.App?.ListFeedTopics) {
        feedTopicsCache = [];
        feedTopicsError = 'Feed 暂不可用';
        renderFeedTopics();
        return;
    }
    if (feedTopicsLoading) return;
    feedTopicsLoading = true;
    feedTopicsError = '';
    renderFeedTopics();
    try {
        const topics = await window.go.app.App.ListFeedTopics();
        feedTopicsCache = Array.isArray(topics) ? topics : [];
        if (!feedTopicsCache.includes(currentFeedTopicId)) {
            currentFeedTopicId = feedTopicsCache[0] || '';
        }
    } catch (error) {
        console.error('加载Feed主题失败:', error);
        feedTopicsCache = [];
        feedTopicsError = '加载Feed主题失败';
    }
    feedTopicsLoading = false;
    renderFeedTopics();
    if (feedTopicsError || feedTopicsCache.length === 0) {
        feedArticlesCache = [];
        feedArticlesLoading = false;
        feedArticlesError = '';
        renderFeedArticles();
        return;
    }
    await loadFeedArticles(currentFeedTopicId);
}

async function loadFeedArticles(topic) {
    if (!feedArticlesList) return;
    if (!topic) {
        feedArticlesCache = [];
        feedArticlesLoading = false;
        feedArticlesError = '';
        renderFeedArticles();
        return;
    }
    if (!window.go?.app?.App?.LoadFeedTopic) {
        feedArticlesCache = [];
        feedArticlesLoading = false;
        feedArticlesError = 'Feed 暂不可用';
        renderFeedArticles();
        return;
    }
    const requestToken = ++feedArticlesRequestToken;
    feedArticlesLoading = true;
    feedArticlesError = '';
    renderFeedArticles();
    try {
        const items = await window.go.app.App.LoadFeedTopic(topic);
        if (requestToken !== feedArticlesRequestToken) return;
        feedArticlesCache = Array.isArray(items) ? items : [];
    } catch (error) {
        if (requestToken !== feedArticlesRequestToken) return;
        console.error('加载Feed内容失败:', error);
        feedArticlesCache = [];
        feedArticlesError = '加载Feed内容失败';
    }
    if (requestToken !== feedArticlesRequestToken) return;
    feedArticlesLoading = false;
    renderFeedArticles();
}

function formatFeedTimestamp(timestamp) {
    if (!timestamp) return '';
    const date = new Date(timestamp);
    if (Number.isNaN(date.getTime())) return '';
    return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        hour12: false
    });
}

function renderFeedDetailMarkdown(content) {
    if (!feedDetailContent) return;
    renderMarkdownInto(feedDetailContent, content);
}

function renderMarkdownInto(target, content) {
    if (!target) return;
    if (!content) {
        target.textContent = '';
        return;
    }
    if (window.marked?.parse && window.DOMPurify?.sanitize) {
        const html = window.marked.parse(content);
        const safeHtml = window.DOMPurify.sanitize(html, { USE_PROFILES: { html: true } });
        target.innerHTML = safeHtml;
        return;
    }
    target.textContent = content;
}
