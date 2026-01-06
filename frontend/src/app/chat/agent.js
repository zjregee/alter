import { dom } from '../dom.js';
import { state } from '../state.js';
import { getEventText, safeParseJSON } from '../utils.js';
import { appendThoughtBlock, appendToolCall, clearToolTimers, hideThinkingStatus, showThinkingStatus, updateToolCall } from './turns.js';
import { appendTurnActions } from './actions.js';
import { handleCancel, handleError } from './feedback.js';
import { setCancelVisibility, setProcessingState } from './processing.js';
import { isCancelMessage } from './message-utils.js';

let refreshThreads = null;

export function setThreadRefresher(refresher) {
    refreshThreads = typeof refresher === 'function' ? refresher : null;
}

export function handleAgentMessage(data) {
    if (state.cancelHandled || !data) return;
    if (state.isLoadingHistory) {
        state.pendingAgentMessages.push({
            data,
            token: state.activeLoadToken,
            threadID: state.activeLoadThreadID
        });
        return;
    }
    processAgentMessage(data);
}

export function processAgentMessage(data) {
    if (state.cancelHandled) return;
    const { type, content } = data;

    if (state.isProcessing && !state.cancelVisible) {
        setCancelVisibility(true);
    }

    switch (type) {
        case 'start_thinking':
            showThinkingStatus();
            break;
        case 'thought': {
            const duration = state.thinkingStartTime ? (Date.now() - state.thinkingStartTime) / 1000 : null;
            hideThinkingStatus();
            appendThoughtBlock(getEventText(content), duration);
            state.lastThinkingDuration = duration;
            break;
        }
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
        case 'error': {
            const errorText = getEventText(content);
            if (isCancelMessage(errorText)) {
                handleCancel();
            } else {
                handleError(errorText);
            }
            break;
        }
    }
}

function handleFinalResponse() {
    hideThinkingStatus();
    clearToolTimers();
    if (state.currentTurnContainer) {
        appendTurnActions(state.currentTurnContainer, true);
    }
    setProcessingState(false);
    dom.chatInput?.focus();
    state.currentTurnContainer = null;
    state.currentToolCallElements.clear();
    state.lastThoughtText = '';
    state.lastThoughtElement = null;
    state.lastThinkingDuration = null;
    if (refreshThreads) {
        refreshThreads();
    }
}
