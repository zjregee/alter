import { state } from '../state.js';
import { setProcessingState } from './processing.js';
import { handleError } from './feedback.js';

export function setTurnRegenerateContext(container, userIndex, userContent) {
    if (!container) return;
    if (typeof userIndex !== 'number' || userIndex < 0) return;
    state.turnRegenerateContext.set(container, {
        userIndex,
        userContent: typeof userContent === 'string' ? userContent : ''
    });
}

export async function handleRegenerateTurn(container) {
    if (!state.currentThreadID || state.isProcessing) return;
    const context = container ? state.turnRegenerateContext.get(container) : null;
    const userIndex = context?.userIndex;
    const userContent = context?.userContent || '';
    if (typeof userIndex !== 'number' || userIndex < 0) {
        handleError('无法定位要重新生成的消息');
        return;
    }
    if (!userContent.trim()) {
        handleError('无法找到要重新生成的内容');
        return;
    }
    setProcessingState(true);
    state.lastThoughtText = '';
    state.lastThoughtElement = null;
    state.lastThinkingDuration = null;
    state.pendingRegenerateContext = {
        userIndex,
        userContent
    };

    try {
        await window.go.app.App.EditAndResendMessage(state.currentThreadID, userContent, userIndex);
    } catch (error) {
        state.pendingRegenerateContext = null;
        console.error('Regenerate response error:', error);
        handleError('抱歉，发生了错误: ' + error);
    }
}
