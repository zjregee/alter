import { dom } from './dom.js';
import { loadFeedTopics } from './feed.js';

let lastActiveView = 'chat';

export function setupViewSwitcher() {
    dom.configBtns.forEach((btn) => {
        btn.addEventListener('click', () => {
            switchView('config');
        });
    });

    if (dom.settingsBackBtn) {
        dom.settingsBackBtn.addEventListener('click', () => {
            switchView(lastActiveView);
        });
    }

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
    const isConfig = viewName === 'config';

    if (!isConfig) {
        lastActiveView = viewName;
    }

    // Toggle Headers
    if (dom.viewSwitcher) {
        dom.viewSwitcher.style.display = isConfig ? 'none' : 'flex';
    }
    if (dom.settingsHeader) {
        dom.settingsHeader.style.display = isConfig ? 'flex' : 'none';
    }

    // Toggle View Content (Sidebar and Main)
    const viewMap = {
        chat: { sidebar: dom.chatViewSidebar, main: dom.chatViewMain },
        notifications: { sidebar: dom.notificationsViewSidebar, main: dom.notificationsViewMain },
        config: { sidebar: dom.settingsSidebar, main: dom.settingsMain }
    };

    Object.entries(viewMap).forEach(([v, containers]) => {
        const isActive = v === viewName;
        if (containers.sidebar) containers.sidebar.classList.toggle('active', isActive);
        if (containers.main) containers.main.classList.toggle('active', isActive);

        // Update view switcher button state
        if (dom.viewSwitcher) {
            const btn = dom.viewSwitcher.querySelector(`.view-btn[data-view="${v}"]`);
            if (btn) btn.classList.toggle('active', isActive);
        }
    });

    if (viewName === 'notifications') {
        loadFeedTopics();
    }
}
