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
    if (!state.currentTurnContainer) return;

    // Remove existing thinking block if any (though usually there isn't one at start)
    if (state.currentThinkingBlock) {
        state.currentThinkingBlock.remove();
    }

    state.currentThinkingBlock = document.createElement('div');
    state.currentThinkingBlock.className = 'thought-block thinking';
    const thinkingBlock = state.currentThinkingBlock;
    
    // Header
    const header = document.createElement('div');
    header.className = 'thought-header';
    header.innerHTML = `
        <span class="tool-icon">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <path d="m12 3-1.9 4.8-4.8 1.9 4.8 1.9 1.9 4.8 1.9-4.8 4.8-1.9-4.8-1.9L12 3z"/>
                <path d="M5 3v4"/>
                <path d="M19 17v4"/>
                <path d="M3 5h4"/>
                <path d="M17 19h4"/>
            </svg>
        </span>
        <span class="thinking-label">Thinking</span>
        <span class="thinking-dots">
            <span class="dot">.</span>
            <span class="dot">.</span>
            <span class="dot">.</span>
        </span>
        <span class="thinking-timer">0.0s</span>
    `;
    
    // Content area for reasoning stream
    const content = document.createElement('div');
    content.className = 'thought-content';
    content.style.display = 'none';
    content.innerHTML = '<div class="markdown-body"></div>';

    // Click to toggle
    header.addEventListener('click', () => {
        thinkingBlock.classList.toggle('expanded');
        content.style.display = thinkingBlock.classList.contains('expanded') ? 'block' : 'none';
    });

    thinkingBlock.appendChild(header);
    thinkingBlock.appendChild(content);

    state.currentTurnContainer.querySelector('.message-content').appendChild(thinkingBlock);

    state.currentThinkingText = '';
    state.thinkingStartTime = Date.now();
    const timerElement = header.querySelector('.thinking-timer');
    state.thinkingTimerInterval = setInterval(() => {
        const elapsed = ((Date.now() - state.thinkingStartTime) / 1000).toFixed(1);
        timerElement.textContent = `${elapsed}s`;
    }, 100);
}

export function updateThinkingChunk(chunk) {
    if (!state.currentThinkingBlock) return;
    
    state.currentThinkingText = (state.currentThinkingText || '') + chunk;
    
    const contentDiv = state.currentThinkingBlock.querySelector('.thought-content');
    const mdBody = contentDiv.querySelector('.markdown-body');
    
    // If we have content, make sure the block indicates it implies 'Process'
    if (state.currentThinkingText.trim() && !state.currentThinkingBlock.classList.contains('has-content')) {
        state.currentThinkingBlock.classList.add('has-content');
        const label = state.currentThinkingBlock.querySelector('.thinking-label');
        if (label) label.textContent = 'Thinking';
    }

    renderMarkdownInto(mdBody, state.currentThinkingText);
    agentScroll();
}

export function updateStreamChunk(chunk) {
    createTurnContainer(); // Ensure container exists
    if (!state.currentTurnContainer) return;

    // Check if we already have a response element
    if (!state.currentResponseElement) {
        state.currentResponseElement = document.createElement('div');
        state.currentResponseElement.className = 'response-block markdown-body';
        state.currentTurnContainer.querySelector('.message-content').appendChild(state.currentResponseElement);
    }

    state.currentResponseText = (state.currentResponseText || '') + chunk;
    renderMarkdownInto(state.currentResponseElement, state.currentResponseText);
    agentScroll();
}

export function hideThinkingStatus() {
    if (state.thinkingTimerInterval) {
        clearInterval(state.thinkingTimerInterval);
        state.thinkingTimerInterval = null;
    }
    // We do NOT remove the block anymore, as it might contain the reasoning trace.
    // Instead we mark it as done/collapsed or finalize it in finalizeTurn.
    // But if it has no content (no reasoning), we might want to remove it or change it to a simple "Thought" label.
    if (state.currentThinkingBlock) {
        // If no text was streamed and it's just a spinner, we might remove it 
        // OR we wait for finalizeTurn to decide.
        // For now, just stop the timer animation.
        const dots = state.currentThinkingBlock.querySelector('.thinking-dots');
        if (dots) dots.style.display = 'none';
        
        state.currentThinkingBlock.classList.remove('thinking');
        state.currentThinkingBlock.classList.add('finished');
    }
}

export function finalizeTurn(thoughtData) {
    hideThinkingStatus(); // Stop timer

    const { reasoning, content, duration_seconds } = thoughtData || {};

    // 1. Handle Reasoning
    if (state.currentThinkingBlock) {
        if (reasoning && reasoning.trim()) {
            // Update with full reasoning text
            state.currentThinkingText = reasoning;
            const mdBody = state.currentThinkingBlock.querySelector('.thought-content .markdown-body');
            renderMarkdownInto(mdBody, reasoning);
            
            // Update Header
            const label = state.currentThinkingBlock.querySelector('.thinking-label');
            if (label) label.textContent = 'Thought';
            
            const timer = state.currentThinkingBlock.querySelector('.thinking-timer');
            if (timer && duration_seconds) {
                timer.textContent = `for ${duration_seconds.toFixed(2)}s`;
            } else if (timer) {
                timer.textContent = '';
            }
            state.currentThinkingBlock.classList.add('has-content');
        } else {
            // No reasoning content: keep the header and timer for parity with previous behavior
            const label = state.currentThinkingBlock.querySelector('.thinking-label');
            if (label) label.textContent = 'Thought';

            const timer = state.currentThinkingBlock.querySelector('.thinking-timer');
            if (timer && duration_seconds) {
                timer.textContent = `for ${duration_seconds.toFixed(2)}s`;
            } else if (timer) {
                timer.textContent = '';
            }

            state.currentThinkingBlock.classList.remove('has-content');
            state.currentThinkingBlock.classList.remove('expanded');
            const contentBlock = state.currentThinkingBlock.querySelector('.thought-content');
            if (contentBlock) contentBlock.remove();
        }
    } else if (reasoning && reasoning.trim()) {
        // If for some reason we didn't have a block (e.g. non-streamed history load), create one
        appendThoughtBlock(reasoning, duration_seconds, true); // true for 'isReasoning'
    }

    // 2. Handle Content (Answer)
    const resolvedContent = typeof content === 'string' ? content : '';
    if (resolvedContent.trim()) {
        if (!state.currentResponseElement) {
            // Create if didn't exist (no stream or just started)
            state.currentResponseElement = document.createElement('div');
            state.currentResponseElement.className = 'response-block markdown-body';
            state.currentTurnContainer.querySelector('.message-content').appendChild(state.currentResponseElement);
        }

        state.currentResponseText = resolvedContent;
        renderMarkdownInto(state.currentResponseElement, resolvedContent);
    } else if (state.currentResponseElement) {
        const existing = state.currentResponseText || state.currentResponseElement.dataset.rawMarkdown || '';
        if (existing.trim()) {
            renderMarkdownInto(state.currentResponseElement, existing);
        } else {
            state.currentResponseElement.remove();
        }
    }

    // Reset temporary state
    state.currentThinkingText = '';
    state.currentResponseText = '';
    state.currentResponseElement = null;
    state.currentThinkingBlock = null;
    
    agentScroll();
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

export function appendThoughtBlock(text, durationInSeconds, isReasoning = false) {
    createTurnContainer();
    if (!state.currentTurnContainer) return;

    const thoughtBlock = document.createElement('div');
    thoughtBlock.className = `thought-block ${isReasoning ? 'thinking finished has-content' : ''}`;

    const durationText = durationInSeconds ? ` for ${durationInSeconds.toFixed(2)}s` : '';
    const label = 'Thought';

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
            <span>${label}<span class="thinking-timer">${durationText}</span></span>
        </div>
        <div class="thought-content" style="${isReasoning ? 'display: none;' : ''}">
            <div class="markdown-body"></div>
        </div>
    `;

    if (isReasoning) {
        thoughtBlock.querySelector('.thought-header').addEventListener('click', () => {
            thoughtBlock.classList.toggle('expanded');
            const content = thoughtBlock.querySelector('.thought-content');
            content.style.display = thoughtBlock.classList.contains('expanded') ? 'block' : 'none';
        });
    }

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
    // Only set these if it's the "answer" part, but appendThoughtBlock is vague now.
    // We'll rely on finalizeTurn for the main flow. This is mostly for history loading or legacy.
    
    state.currentTurnContainer.querySelector('.message-content').appendChild(thoughtBlock);
    agentScroll();
}

export function appendResponseBlock(text) {
    createTurnContainer();
    if (!state.currentTurnContainer) return;

    const resolvedText = typeof text === 'string' ? text : String(text ?? '');
    if (!resolvedText.trim()) return;

    const responseBlock = document.createElement('div');
    responseBlock.className = 'response-block markdown-body';
    renderMarkdownInto(responseBlock, resolvedText);
    state.currentTurnContainer.querySelector('.message-content').appendChild(responseBlock);
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
