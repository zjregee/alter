export function normalizeEventContent(content) {
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

export function safeParseJSON(text) {
    try {
        const normalized = normalizeEventContent(text);
        if (normalized && typeof normalized === 'object') {
            return normalized;
        }
        return JSON.parse(normalized);
    } catch {
        return null;
    }
}

export function formatToolArgs(args) {
    if (args == null) return '';
    if (typeof args === 'string') {
        try {
            const parsed = JSON.parse(args);
            if (parsed && typeof parsed === 'object') {
                return JSON.stringify(parsed, null, 4);
            }
        } catch {
            return args;
        }
        return args;
    }
    return JSON.stringify(args, null, 4);
}

export function getEventText(content) {
    const payload = safeParseJSON(content);
    if (payload && typeof payload === 'object') {
        if (typeof payload.content === 'string') return payload.content;
        if (typeof payload.error === 'string') return payload.error;
    }
    return typeof content === 'string' ? content : '';
}

export function getEventDurationSeconds(payload) {
    if (!payload || typeof payload !== 'object') return null;
    const duration = payload.duration_seconds;
    if (typeof duration === 'number' && Number.isFinite(duration)) {
        return duration;
    }
    return null;
}

export function formatFeedTimestamp(timestamp) {
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

export function formatMessage(content) {
    if (!content || typeof content !== 'string') {
        return content;
    }

    // Ported from internal/app/formatter.go
    // Note: JS requires 'u' flag for Unicode property escapes like \p{Unified_Ideograph}
    // \p{Unified_Ideograph} is the JS equivalent of \p{Han}
    let formatted = content;

    // hanToLatinMidPunct: ([\p{Han}])([-/]+)([A-Za-z0-9]) -> $1 $2 $3
    formatted = formatted.replace(/(\p{Unified_Ideograph})([-/]+)([A-Za-z0-9])/gu, '$1 $2 $3');
    // latinToHanMidPunct: ([A-Za-z0-9])([-/]+)([\p{Han}]) -> $1 $2 $3
    formatted = formatted.replace(/([A-Za-z0-9])([-/]+)(\p{Unified_Ideograph})/gu, '$1 $2 $3');

    // hanToLatinOpenPunct: ([\p{Han}])([\(\[\{'""]+)([A-Za-z0-9]) -> $1 $2$3
    formatted = formatted.replace(/(\p{Unified_Ideograph})([([{'""]+)([A-Za-z0-9])/gu, '$1 $2$3');
    // latinToHanOpenPunct: ([A-Za-z0-9])([\(\[\{'""]+)([\p{Han}]) -> $1 $2$3
    formatted = formatted.replace(/([A-Za-z0-9])([([{'""]+)(\p{Unified_Ideograph})/gu, '$1 $2$3');

    // hanToLatinPunct: ([\p{Han}])([,.;:!?\)\]\}]+)([A-Za-z0-9]) -> $1$2 $3
    formatted = formatted.replace(/(\p{Unified_Ideograph})([,.;:!?)}]+)([A-Za-z0-9])/gu, '$1$2 $3');
    // latinToHanPunct: ([A-Za-z0-9])([,.;:!?\)\]\}]+)([\p{Han}]) -> $1$2 $3
    formatted = formatted.replace(/([A-Za-z0-9])([,.;:!?)}]+)(\p{Unified_Ideograph})/gu, '$1$2 $3');

    // hanToLatin: ([\p{Han}])([A-Za-z0-9]) -> $1 $2
    formatted = formatted.replace(/(\p{Unified_Ideograph})([A-Za-z0-9])/gu, '$1 $2');
    // latinToHan: ([A-Za-z0-9])([\p{Han}]) -> $1 $2
    formatted = formatted.replace(/([A-Za-z0-9])(\p{Unified_Ideograph})/gu, '$1 $2');

    return formatted;
}

export function renderMarkdownInto(target, content) {
    if (!target) return;
    if (!content) {
        target.textContent = '';
        return;
    }
    const formattedContent = formatMessage(content);
    target.dataset.rawMarkdown = typeof content === 'string' ? content : String(content);
    if (window.marked?.parse && window.DOMPurify?.sanitize) {
        const html = window.marked.parse(formattedContent);
        const safeHtml = window.DOMPurify.sanitize(html, { USE_PROFILES: { html: true } });
        target.innerHTML = safeHtml;
        return;
    }
    target.textContent = formattedContent;
}
