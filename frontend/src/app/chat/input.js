import { dom } from '../dom.js';
import { state } from '../state.js';
import { createNewThread } from '../threads.js';
import { handleError } from './feedback.js';
import { markCanceling, resetCancelButton, setProcessingState } from './processing.js';
import { appendStatusMessage, addUserMessage } from './turns.js';
import { handleCancel } from './feedback.js';

export function setupChatInputHandlers() {
    if (dom.chatInput) {
        dom.chatInput.addEventListener('input', function() {
            this.style.height = 'auto';
            this.style.height = Math.min(this.scrollHeight, 300) + 'px';
        });
    }

    if (dom.inputWrapper && dom.chatInput) {
        dom.inputWrapper.addEventListener('click', (e) => {
            if (e.target.closest('button') || e.target.closest('a') || e.target === dom.chatInput) {
                return;
            }
            dom.chatInput.focus();
        });
    }

    dom.sendBtn?.addEventListener('click', handleSendMessage);
    dom.cancelBtn?.addEventListener('click', handleCancelClick);

    dom.chatInput?.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            if (state.isComposing || e.isComposing || e.keyCode === 229 || state.suppressEnterOnce) {
                e.preventDefault();
                state.suppressEnterOnce = false;
                return;
            }
            e.preventDefault();
            dom.sendBtn?.click();
        }
    });

    dom.chatInput?.addEventListener('compositionstart', () => {
        state.isComposing = true;
    });

    dom.chatInput?.addEventListener('compositionend', () => {
        state.isComposing = false;
        state.suppressEnterOnce = true;
        setTimeout(() => {
            state.suppressEnterOnce = false;
        }, 0);
    });

    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && state.isProcessing && state.cancelVisible && !dom.cancelBtn?.disabled) {
            e.preventDefault();
            dom.cancelBtn?.click();
        }
    });
}

async function handleSendMessage() {
    const message = dom.chatInput?.value.trim();
    if (!message || state.isProcessing) {
        return;
    }
    if (!state.currentThreadID) {
        await createNewThread();
    }

    addUserMessage(message, true);
    state.userMessageCache[state.userMessageCount] = message;
    state.pendingRegenerateContext = {
        userIndex: state.userMessageCount,
        userContent: message
    };
    state.userMessageCount += 1;
    if (dom.chatInput) {
        dom.chatInput.value = '';
        dom.chatInput.style.height = 'auto';
    }

    setProcessingState(true);
    state.lastThoughtText = '';
    state.lastThoughtElement = null;
    state.lastThinkingDuration = null;

    try {
        await window.go.app.App.AgentChat(state.currentThreadID, message);
    } catch (error) {
        console.error('Agent chat error:', error);
        handleError('抱歉，发生了错误: ' + error);
    }
}

async function handleCancelClick() {
    if (!state.currentThreadID || !state.isProcessing) return;
    markCanceling();
    try {
        await window.go.app.App.CancelStreamRequestToThread(state.currentThreadID);
        if (!state.cancelHandled) {
            handleCancel();
        }
    } catch (error) {
        console.error('Cancel agent request error:', error);
        resetCancelButton();
        appendStatusMessage(state.currentTurnContainer, '取消失败: ' + error, 'var(--error-text)');
    }
}
