export function isCancelMessage(message) {
    if (typeof message !== 'string') return false;
    return (
        message.includes('agent generation cancelled') ||
        message.includes('context canceled') ||
        message.includes('context cancelled')
    );
}
