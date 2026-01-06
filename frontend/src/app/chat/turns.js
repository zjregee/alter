import { dom } from '../dom.js';
import { state } from '../state.js';
import { formatToolArgs, getEventDurationSeconds, renderMarkdownInto } from '../utils.js';
import { setTurnRegenerateContext } from './regenerate.js';

export function agentScroll() {
    if (!dom.chatMessages) return;
    dom.chatMessages.scrollTop = dom.chatMessages.scrollHeight;
}

export function setChatEmptyState(visible) {
    if (!dom.chatMessages) return;
    dom.chatMessages.classList.toggle('is-empty', visible);
    const existing = dom.chatMessages.querySelector('.chat-empty');
    if (!visible) {
        existing?.remove();
        return;
    }
    if (existing) return;
    const empty = document.createElement('div');
    empty.className = 'chat-empty';
    empty.innerHTML = `
        <div class="chat-empty-title">Start a new chat</div>
        <div class="chat-empty-sub">Try: summarize files, triage issues, or run a skill.</div>
    `;
    dom.chatMessages.appendChild(empty);
}

export function createTurnContainer() {
    if (state.currentTurnContainer || !dom.chatMessages) return;
    setChatEmptyState(false);
    const turnContainer = document.createElement('div');
    turnContainer.className = 'message-group assistant-message';

    const handleMouseEnter = () => {
        turnContainer.classList.add('is-hovered');
    };

    const handleMouseLeave = (e) => {
        const relatedTarget = e.relatedTarget;

        if (relatedTarget) {
            const actions = turnContainer.querySelector('.message-actions');
            if (actions && (relatedTarget === actions || actions.contains(relatedTarget))) {
                return;
            }
        }

        if (turnContainer.querySelector('.more-menu.is-open')) {
            return;
        }

        turnContainer.classList.remove('is-hovered');
    };

    turnContainer.addEventListener('mouseenter', handleMouseEnter);
    turnContainer.addEventListener('mouseleave', handleMouseLeave);

    const messageContent = document.createElement('div');
    messageContent.className = 'message-content';
    turnContainer.appendChild(messageContent);
    dom.chatMessages.appendChild(turnContainer);
    state.currentTurnContainer = turnContainer;

    if (!state.isLoadingHistory && state.pendingRegenerateContext) {
        setTurnRegenerateContext(
            state.currentTurnContainer,
            state.pendingRegenerateContext.userIndex,
            state.pendingRegenerateContext.userContent
        );
        state.pendingRegenerateContext = null;
    }
}

export function showThinkingStatus() {
    createTurnContainer();

    if (state.currentThinkingBlock || !state.currentTurnContainer) return;

    state.currentThinkingBlock = document.createElement('div');
    state.currentThinkingBlock.className = 'thinking-status';
    state.currentThinkingBlock.innerHTML = `
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
    state.currentThinkingBlock.style.display = 'flex';

    state.currentTurnContainer.querySelector('.message-content').appendChild(state.currentThinkingBlock);

    const timerElement = state.currentThinkingBlock.querySelector('.thinking-timer');
    state.thinkingStartTime = Date.now();
    state.thinkingTimerInterval = setInterval(() => {
        const elapsed = ((Date.now() - state.thinkingStartTime) / 1000).toFixed(1);
        timerElement.textContent = `${elapsed}s`;
    }, 100);
}

export function hideThinkingStatus() {
    if (state.thinkingTimerInterval) {
        clearInterval(state.thinkingTimerInterval);
        state.thinkingTimerInterval = null;
    }
    if (state.currentThinkingBlock) {
        state.currentThinkingBlock.remove();
        state.currentThinkingBlock = null;
    }
}

export function clearToolTimers() {
    for (const intervalId of state.toolTimerIntervals.values()) {
        clearInterval(intervalId);
    }
    state.toolTimerIntervals.clear();
}

export function startToolTimer(toolCallEl, toolId) {
    const durationEl = toolCallEl.querySelector('.duration');
    if (!durationEl) return;
    durationEl.textContent = '0.0s';
    const startTime = parseInt(toolCallEl.dataset.startTime, 10);
    const intervalId = setInterval(() => {
        const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
        durationEl.textContent = `${elapsed}s`;
    }, 100);
    state.toolTimerIntervals.set(toolId, intervalId);
}

export function stopToolTimer(toolId) {
    const intervalId = state.toolTimerIntervals.get(toolId);
    if (intervalId) {
        clearInterval(intervalId);
    }
    state.toolTimerIntervals.delete(toolId);
}

export function appendThoughtBlock(text, durationInSeconds) {
    createTurnContainer();
    if (!state.currentTurnContainer) return;

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
    const trimmedText = typeof displayText === 'string' ? displayText.trim() : String(displayText).trim();
    if (!trimmedText) {
        thoughtBlock.classList.add('is-empty');
        const content = thoughtBlock.querySelector('.thought-content');
        if (content) {
            content.style.display = 'none';
        }
    }
    thoughtBlock.dataset.rawMarkdown = typeof displayText === 'string' ? displayText : String(displayText);
    if (trimmedText) {
        renderMarkdownInto(thoughtBlock.querySelector('.markdown-body'), displayText);
    }
    state.lastThoughtText = trimmedText;
    state.lastThoughtElement = thoughtBlock;

    state.currentTurnContainer.querySelector('.message-content').appendChild(thoughtBlock);
    agentScroll();
}

export function addUserMessage(text, scrollToBottom = true) {
    if (!dom.chatMessages) return;
    setChatEmptyState(false);
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

    dom.chatMessages.appendChild(messageGroup);
    if (scrollToBottom) {
        requestAnimationFrame(() => {
            const messageTop = messageGroup.getBoundingClientRect().top;
            const containerTop = dom.chatMessages.getBoundingClientRect().top;
            dom.chatMessages.scrollTo({
                top: dom.chatMessages.scrollTop + (messageTop - containerTop),
                behavior: 'smooth'
            });
        });
    }
}

export function appendStatusMessage(container, text, color) {
    if (!container) return;
    const content = container.querySelector('.message-content');
    if (!content) return;
    const status = document.createElement('div');
    status.className = 'response-meta status-message';
    status.style.color = color;
    const span = document.createElement('span');
    span.textContent = text;
    status.appendChild(span);
    content.appendChild(status);
}

export function appendToolCall(payload, isHistory = false) {
    if (!payload || typeof payload !== 'object' || !payload.id) return;
    createTurnContainer();
    if (!state.currentTurnContainer) return;

    const toolCallsContainer = state.currentTurnContainer.querySelector('.message-content');
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
    state.currentToolCallElements.set(payload.id, toolCallEl);
    if (!isHistory) {
        startToolTimer(toolCallEl, payload.id);
    }
    agentScroll();
}

export function updateToolCall(payload, isHistory = false) {
    if (!payload || typeof payload !== 'object' || !payload.id || !state.currentToolCallElements.has(payload.id)) return;

    const toolCallEl = state.currentToolCallElements.get(payload.id);
    stopToolTimer(payload.id);

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

    if (!isHistory) {
        state.currentToolCallElements.delete(payload.id);
    }
    agentScroll();
}
