import { dom } from './dom.js';

export function setupSidebarResizer() {
    if (!dom.sidebarResizer || !dom.sidebar) return;
    dom.sidebarResizer.addEventListener('mousedown', (e) => {
        e.preventDefault();
        const startX = e.clientX;
        const startWidth = dom.sidebar.getBoundingClientRect().width;
        const onMouseMove = (moveEvent) => {
            if (dom.sidebar.classList.contains('is-collapsed')) return;
            const nextWidth = Math.min(360, Math.max(180, startWidth + (moveEvent.clientX - startX)));
            dom.sidebar.style.width = `${Math.round(nextWidth)}px`;
        };
        const onMouseUp = () => {
            document.removeEventListener('mousemove', onMouseMove);
            document.removeEventListener('mouseup', onMouseUp);
            document.body.style.cursor = '';
            document.body.style.userSelect = '';
        };
        document.addEventListener('mousemove', onMouseMove);
        document.addEventListener('mouseup', onMouseUp);
        document.body.style.cursor = 'col-resize';
        document.body.style.userSelect = 'none';
    });
}

export function setupSidebarToggle() {
    if (!dom.sidebarToggles || !dom.sidebar) return;

    dom.sidebarToggles.forEach((btn) => {
        btn.addEventListener('click', () => {
            dom.sidebar.classList.toggle('is-collapsed');
        });
    });
}
