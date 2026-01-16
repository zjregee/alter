import { dom } from './dom.js';
import { loadSchedules } from './scheduler.js';
import { loadUsageData } from './usage.js';

export function setupSettingsNavigation() {
    if (!dom.settingsSidebar) return;

    dom.settingsSidebar.addEventListener('click', (e) => {
        const item = e.target.closest('.settings-item');
        if (!item) return;

        const section = item.dataset.section;
        if (!section) return;

        // Update active state in sidebar
        dom.settingsSidebar.querySelectorAll('.settings-item').forEach((btn) => {
            btn.classList.toggle('active', btn === item);
        });

        // Show corresponding content
        if (dom.settingsMain) {
            dom.settingsMain.querySelectorAll('.settings-content').forEach((content) => {
                content.classList.toggle('active', content.id === `${section}-settings`);
            });
        }

        // Load specific section data
        if (section === 'scheduler') {
            loadSchedules();
        } else if (section === 'usage') {
            loadUsageData();
        }
    });
}

export function onSettingsViewActivated() {
    const activeItem = dom.settingsSidebar?.querySelector('.settings-item.active');
    if (activeItem) {
        if (activeItem.dataset.section === 'scheduler') {
            loadSchedules();
        } else if (activeItem.dataset.section === 'usage') {
            loadUsageData();
        }
    }
}
