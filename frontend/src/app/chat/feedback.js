import { dom } from '../dom.js';
import { state } from '../state.js';
import { appendStatusMessage, clearToolTimers, createTurnContainer, hideThinkingStatus } from './turns.js';
import { setProcessingState } from './processing.js';

export function handleCancel() {
    state.cancelHandled = true;
    hideThinkingStatus();
    clearToolTimers();
    if (!state.currentTurnContainer) {
        createTurnContainer();
    }
    appendStatusMessage(state.currentTurnContainer, 'Cancelled', 'var(--text-tertiary)');

    setProcessingState(false);
    dom.chatInput?.focus();
    state.currentTurnContainer = null;
    state.currentToolCallElements.clear();
    state.lastThoughtText = '';
    state.lastThoughtElement = null;
    state.lastThinkingDuration = null;
    state.pendingStreamBuffer = '';
    state.isStreamingLoopActive = false;
    state.pendingThinkingBuffer = '';
    state.isThinkingLoopActive = false;
}

export function handleError(errorMessage) {
    hideThinkingStatus();
    clearToolTimers();
    if (!state.currentTurnContainer) {
        createTurnContainer();
    }
    appendStatusMessage(state.currentTurnContainer, '错误: ' + errorMessage, 'var(--error-text)');

    setProcessingState(false);
    dom.chatInput?.focus();
    state.currentTurnContainer = null;
    state.currentToolCallElements.clear();
    state.lastThoughtText = '';
    state.lastThoughtElement = null;
    state.lastThinkingDuration = null;
    state.pendingStreamBuffer = '';
    state.isStreamingLoopActive = false;
    state.pendingThinkingBuffer = '';
    state.isThinkingLoopActive = false;
}
