import { dom } from '../dom.js';
import { handleRegenerateTurn } from './regenerate.js';

export function setupActionMenuListeners() {
    document.addEventListener('click', (e) => {
        if (!e.target.closest('.more-menu')) {
            closeAllActionMenus();
        }
    });

    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            closeAllActionMenus();
        }
    });

    dom.chatMessages?.addEventListener('scroll', () => {
        closeAllActionMenus();
    });

    window.addEventListener('resize', () => {
        closeAllActionMenus();
    });
}

export function appendTurnActions(container, allowRegenerate) {
    if (!container) return;
    if (container.querySelector('.message-actions')) return;

    const actions = document.createElement('div');
    actions.className = 'message-actions';

    actions.addEventListener('mouseenter', () => {
        container.classList.add('is-hovered');
    });

    actions.addEventListener('mouseleave', (e) => {
        const relatedTarget = e.relatedTarget;

        if (relatedTarget && (relatedTarget === container || container.contains(relatedTarget))) {
            return;
        }

        if (container.querySelector('.more-menu.is-open')) {
            return;
        }

        container.classList.remove('is-hovered');
    });

    const copyBtn = document.createElement('button');
    copyBtn.className = 'icon-btn';
    copyBtn.title = '复制';
    copyBtn.innerHTML = `
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <rect x="9" y="9" width="13" height="13" rx="2"></rect>
            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
        </svg>
    `;
    copyBtn.addEventListener('click', () => handleCopyTurn(container, copyBtn));

    const regenBtn = document.createElement('button');
    regenBtn.className = 'icon-btn';
    regenBtn.title = allowRegenerate ? '重新生成' : '仅支持已完成回复重新生成';
    regenBtn.innerHTML = `
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <polyline points="18 2 22 5 18 8"></polyline>
            <path d="M2 10.5V9a4 4 0 0 1 4-4h16"></path>
            <polyline points="6 22 2 19 6 16"></polyline>
            <path d="M22 13.5v2a4 4 0 0 1-4 4H2"></path>
        </svg>
    `;
    if (!allowRegenerate) {
        regenBtn.classList.add('is-disabled');
    } else {
        regenBtn.addEventListener('click', () => handleRegenerateTurn(container));
    }

    actions.appendChild(copyBtn);
    actions.appendChild(regenBtn);
    actions.appendChild(createMoreMenu(container));
    container.querySelector('.message-content').appendChild(actions);
}

function closeAllActionMenus(except) {
    document.querySelectorAll('.more-menu.is-open').forEach((menu) => {
        if (except && menu === except) return;
        menu.classList.remove('is-open');
        const messageGroup = menu.closest('.message-group.assistant-message');
        if (messageGroup && !messageGroup.matches(':hover')) {
            messageGroup.classList.remove('is-hovered');
        }
    });
}

function positionMoreMenuList(menu, trigger) {
    const list = menu.querySelector('.more-menu-list');
    if (!list || !trigger) return;
    const rect = trigger.getBoundingClientRect();
    const width = list.offsetWidth || 160;
    const padding = 8;
    const gap = 6;
    const viewportWidth = document.documentElement.clientWidth;
    const viewportHeight = document.documentElement.clientHeight;
    const minLeft = padding;
    const maxLeft = viewportWidth - padding;
    const minTop = padding;
    const maxTop = viewportHeight - padding;

    let left = rect.right - width;
    let top = rect.bottom + gap;

    if (left < minLeft) {
        left = minLeft;
    } else if (left + width > maxLeft) {
        left = Math.max(minLeft, maxLeft - width);
    }

    if (top < minTop) {
        top = minTop;
    }

    const availableBelow = Math.max(0, maxTop - top);
    list.style.maxHeight = `${Math.max(availableBelow, 0)}px`;
    list.style.left = `${Math.round(left)}px`;
    list.style.top = `${Math.round(top)}px`;
}

function createMoreMenu(container) {
    const menu = document.createElement('div');
    menu.className = 'more-menu';

    const moreBtn = document.createElement('button');
    moreBtn.className = 'icon-btn';
    moreBtn.title = '更多';
    moreBtn.innerHTML = `
        <svg width="13" height="13" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <circle cx="5" cy="12" r="1.8"></circle>
            <circle cx="12" cy="12" r="1.8"></circle>
            <circle cx="19" cy="12" r="1.8"></circle>
        </svg>
    `;
    moreBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        const willOpen = !menu.classList.contains('is-open');
        closeAllActionMenus(menu);
        if (willOpen) {
            positionMoreMenuList(menu, moreBtn);
        }
        menu.classList.toggle('is-open', willOpen);
    });

    const list = document.createElement('div');
    list.className = 'more-menu-list';

    const plainItem = document.createElement('button');
    plainItem.type = 'button';
    plainItem.className = 'more-menu-item';
    plainItem.textContent = 'Copy as plain text';
    plainItem.addEventListener('click', (e) => {
        e.stopPropagation();
        handleCopyTurn(container, moreBtn);
        menu.classList.remove('is-open');
    });

    const markdownItem = document.createElement('button');
    markdownItem.type = 'button';
    markdownItem.className = 'more-menu-item';
    markdownItem.textContent = 'Copy as markdown';
    markdownItem.addEventListener('click', (e) => {
        e.stopPropagation();
        handleCopyTurnMarkdown(container, moreBtn);
        menu.classList.remove('is-open');
    });

    list.appendChild(plainItem);
    list.appendChild(markdownItem);
    menu.appendChild(moreBtn);
    menu.appendChild(list);
    return menu;
}

function getTurnCopyText(container) {
    if (!container) return '';
    const parts = [];
    const blocks = container.querySelectorAll('.thought-content, .message-content > p');
    blocks.forEach((block) => {
        const text = block.innerText?.trim();
        if (text) parts.push(text);
    });
    return parts.join('\n\n');
}

function getTurnCopyMarkdown(container) {
    if (!container) return '';
    const parts = [];
    const thoughtBlocks = container.querySelectorAll('.thought-block');
    thoughtBlocks.forEach((block) => {
        const raw = block.dataset.rawMarkdown;
        const text = typeof raw === 'string' ? raw.trim() : block.textContent?.trim();
        if (text) parts.push(text);
    });

    const messageContent = container.querySelector('.message-content');
    let rawMessage = messageContent?.dataset.rawMarkdown || '';
    if (rawMessage && rawMessage.trim()) {
        parts.push(rawMessage.trim());
    } else {
        const rawBlocks = messageContent
            ? Array.from(messageContent.querySelectorAll('[data-raw-markdown]')).filter(
                  (node) => !node.closest('.thought-block')
              )
            : [];
        if (rawBlocks.length) {
            rawBlocks.forEach((block) => {
                const raw = block.dataset.rawMarkdown;
                const text = typeof raw === 'string' ? raw.trim() : block.textContent?.trim();
                if (text) parts.push(text);
            });
        } else {
            const blocks = container.querySelectorAll('.message-content > p');
            blocks.forEach((block) => {
                const text = block.textContent?.trim();
                if (text) parts.push(text);
            });
        }
    }
    return parts.join('\n\n');
}

async function copyTextToClipboard(text) {
    if (!text) return false;
    try {
        await navigator.clipboard.writeText(text);
        return true;
    } catch {
        const textarea = document.createElement('textarea');
        textarea.value = text;
        textarea.style.position = 'fixed';
        textarea.style.left = '-9999px';
        document.body.appendChild(textarea);
        textarea.select();
        const copied = document.execCommand('copy');
        textarea.remove();
        return copied;
    }
}

async function handleCopyTurn(container, button) {
    const copied = await copyTextToClipboard(getTurnCopyText(container));
    if (copied) {
        showCopyFeedback(button);
    }
}

async function handleCopyTurnMarkdown(container, button) {
    const copied = await copyTextToClipboard(getTurnCopyMarkdown(container));
    if (copied) {
        showCopyFeedback(button);
    }
}

function showCopyFeedback(button) {
    if (!button) return;
    if (!button.dataset.originalIcon) {
        button.dataset.originalIcon = button.innerHTML;
    }
    if (!button.dataset.originalTitle) {
        button.dataset.originalTitle = button.title || '';
    }
    if (button._copyTimer) {
        clearTimeout(button._copyTimer);
    }
    button.classList.add('is-copied');
    button.title = '已复制';
    button.innerHTML = `
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <polyline points="20 6 9 17 4 12"></polyline>
        </svg>
    `;
    button._copyTimer = setTimeout(() => {
        if (!button.isConnected) return;
        button.classList.remove('is-copied');
        button.title = button.dataset.originalTitle || '';
        if (button.dataset.originalIcon) {
            button.innerHTML = button.dataset.originalIcon;
        }
    }, 500);
}
