import { dom } from '../dom.js';
import { state } from '../state.js';
import { updateWorkspaceToggleState } from '../workspaces.js';
import { updateModelToggleState } from '../models.js';

export function setProcessingState(nextState) {
    state.isProcessing = nextState;
    if (dom.chatInput) dom.chatInput.disabled = nextState;
    if (dom.sendBtn) dom.sendBtn.disabled = nextState;
    setCancelVisibility(nextState);
    if (nextState) {
        state.cancelHandled = false;
    }
    updateWorkspaceToggleState();
    updateModelToggleState();
}

export function setCancelVisibility(visible) {
    if (!dom.cancelBtn) return;
    state.cancelVisible = visible;
    if (dom.sendCancelStack) {
        dom.sendCancelStack.classList.toggle('cancel-visible', visible);
    }
    dom.cancelBtn.hidden = !visible;
    dom.cancelBtn.disabled = !visible;
    dom.cancelBtn.classList.remove('is-canceling');
}

export function markCanceling() {
    if (!dom.cancelBtn) return;
    dom.cancelBtn.disabled = true;
    dom.cancelBtn.classList.add('is-canceling');
}

export function resetCancelButton() {
    if (!dom.cancelBtn) return;
    dom.cancelBtn.disabled = false;
    dom.cancelBtn.classList.remove('is-canceling');
}
