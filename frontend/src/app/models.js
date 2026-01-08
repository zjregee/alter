import { dom } from './dom.js';
import { state } from './state.js';

export async function loadModels() {
    if (!window.go?.app?.App?.ListModels) return;
    try {
        state.modelsCache = await window.go.app.App.ListModels();
    } catch (error) {
        console.error('加载模型列表失败:', error);
        state.modelsCache = [];
    }
    renderModelList();
    updateModelToggle();
    updateModelToggleState();
}

export function updateModelToggle() {
    if (!dom.modelToggleLabel) return;
    const currentModel = state.modelsCache.find((model) => model.id === state.currentModelID);
    dom.modelToggleLabel.textContent = currentModel ? currentModel.name : 'Select model';
}

export function updateModelToggleState() {
    if (!dom.modelToggle) return;
    dom.modelToggle.disabled = !state.currentThreadID || state.isProcessing || state.modelsCache.length === 0;
}

export function syncCurrentThreadModel() {
    const thread = state.threadsCache.find((item) => item.id === state.currentThreadID);
    state.currentModelID = thread ? thread.model : '';
    updateModelToggle();
    renderModelList();
    updateModelToggleState();
}

export function setModelDropdownOpen(isOpen) {
    if (!dom.modelSwitcher || !dom.modelDropdown || !dom.modelToggle) return;
    dom.modelSwitcher.classList.toggle('open', isOpen);
    dom.modelToggle.setAttribute('aria-expanded', String(isOpen));
    dom.modelDropdown.setAttribute('aria-hidden', String(!isOpen));
    if (isOpen && dom.modelSearchInput) dom.modelSearchInput.focus();
}

export function renderModelList() {
    if (!dom.modelList) return;
    dom.modelList.innerHTML = '';
    const term = state.modelSearchTerm.toLowerCase();
    const filtered = state.modelsCache.filter(
        (m) =>
            !term ||
            m.name.toLowerCase().includes(term) ||
            m.id.toLowerCase().includes(term) ||
            (m.provider || '').toLowerCase().includes(term)
    );

    if (filtered.length === 0) {
        dom.modelList.innerHTML = `<div class="model-empty">${state.modelsCache.length === 0 ? 'No models available' : 'No models found'}</div>`;
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
        models.forEach((model) => {
            const option = document.createElement('button');
            option.type = 'button';
            option.className = 'model-option';
            if (model.id === state.currentModelID) option.classList.add('selected');
            option.innerHTML = `<span class="model-option-name">${model.name}</span><span class="model-option-meta">${model.context_window}</span>`;
            option.addEventListener('click', () => selectModel(model.id));
            groupEl.appendChild(option);
        });
        dom.modelList.appendChild(groupEl);
    });
}

async function selectModel(modelID) {
    if (!state.currentThreadID || state.isProcessing || modelID === state.currentModelID) {
        setModelDropdownOpen(false);
        return;
    }
    try {
        await window.go.app.App.UpdateThreadModel(state.currentThreadID, modelID);
        state.currentModelID = modelID;
        const thread = state.threadsCache.find((item) => item.id === state.currentThreadID);
        if (thread) thread.model = modelID;
        updateModelToggle();
        renderModelList();
        setModelDropdownOpen(false);
    } catch (error) {
        console.error('切换模型失败:', error);
    }
}

export function setupModelHandlers() {
    if (dom.modelSearchInput) {
        dom.modelSearchInput.addEventListener('input', () => {
            state.modelSearchTerm = dom.modelSearchInput.value.trim().toLowerCase();
            renderModelList();
        });
        dom.modelSearchInput.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                setModelDropdownOpen(false);
                dom.modelToggle?.focus();
            }
        });
    }
}
