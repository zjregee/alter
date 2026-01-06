import { dom } from './dom.js';
import { loadFeedTopics } from './feed.js';

export function setupViewSwitcher() {
    if (!dom.viewSwitcher) return;
    dom.viewSwitcher.addEventListener('click', (e) => {
        const viewBtn = e.target.closest('.view-btn');
        if (!viewBtn) return;

        const view = viewBtn.dataset.view;
        if (view) {
            switchView(view);
        }
    });
}

function switchView(viewName) {
    document.querySelectorAll('.view-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.view === viewName);
    });

    document.querySelectorAll('.sidebar-content .view-content').forEach(content => {
        content.classList.toggle('active', content.id === `${viewName}-view-sidebar`);
    });

    document.querySelectorAll('.main-content .view-content').forEach(content => {
        content.classList.toggle('active', content.id === `${viewName}-view-main`);
    });

    if (viewName === 'notifications') {
        loadFeedTopics();
    }
}
