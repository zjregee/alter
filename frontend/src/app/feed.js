import { dom } from './dom.js';
import { state } from './state.js';
import { formatFeedTimestamp, renderMarkdownInto } from './utils.js';

export function setupFeedHandlers() {
    if (window.marked?.setOptions) {
        window.marked.setOptions({
            gfm: true,
            breaks: true
        });
    }

    dom.feedBackBtn?.addEventListener('click', () => {
        if (dom.feedDetailView) dom.feedDetailView.style.display = 'none';
        if (dom.feedArticlesList) dom.feedArticlesList.style.display = 'block';
    });
}

export function handleFeedItemPushed(item) {
    const topic = item?.topic;
    if (!topic) return;

    // 1. Update List UI immediately if it's the current topic
    if (topic === state.currentFeedTopicId) {
        let updated = false;
        if (item?.id) {
            const existingIndex = state.feedArticlesCache.findIndex((article) => article?.id === item.id);
            if (existingIndex !== -1) {
                state.feedArticlesCache[existingIndex] = item;
                updated = true;
            }
        }
        if (!updated) {
            state.feedArticlesCache = [item, ...state.feedArticlesCache];
            state.feedArticlesCache.sort((a, b) => (b?.created_at || 0) - (a?.created_at || 0));
        }
        renderFeedArticles();
    }

    // 2. Incremental update for unread counts instead of full refreshTopicStatuses
    // This avoids the "flicker" of reloading the entire topic list
    if (!item.is_read) {
        // If it's a brand new topic, we still need a full refresh to show it in the sidebar
        if (!state.feedTopicsCache.includes(topic)) {
            refreshTopicStatuses();
        } else {
            // Just increment the count and update the specific dot in DOM
            state.feedTopicsUnread[topic] = (state.feedTopicsUnread[topic] || 0) + 1;
            updateTopicDotUI(topic);
            updateGlobalUnreadDot();
        }
    }
}

function updateTopicDotUI(topic) {
    const topicEl = document.querySelector(`.feed-topic-item[data-topic-id="${CSS.escape(topic)}"]`);
    if (topicEl) {
        const count = state.feedTopicsUnread[topic] || 0;
        if (count > 0) {
            topicEl.classList.add('has-unread');
        } else {
            topicEl.classList.remove('has-unread');
        }
    }
}

async function refreshTopicStatuses() {
    if (!window.go?.app?.App?.ListFeedTopicStatuses) return;
    try {
        const statuses = await window.go.app.App.ListFeedTopicStatuses();
        if (Array.isArray(statuses)) {
            // Update cache of topics if needed (for sidebar list)
            // We want to preserve order if possible or just rely on backend sort
            const newTopics = statuses.map((s) => s.name);
            // Simple check if list changed significantly, or just overwrite
            state.feedTopicsCache = newTopics;

            // Update unread counts
            state.feedTopicsUnread = {};
            statuses.forEach((s) => {
                state.feedTopicsUnread[s.name] = s.unread_count;
            });

            updateGlobalUnreadDot();
            renderFeedTopics();
        }
    } catch (err) {
        console.error('Failed to refresh topic statuses', err);
    }
}

export function renderFeedTopics() {
    if (!dom.feedTopicsList) return;
    dom.feedTopicsList.innerHTML = '';
    if (state.feedTopicsLoading) {
        dom.feedTopicsList.innerHTML = '<div class="feed-empty">Loading topics...</div>';
        return;
    }
    if (state.feedTopicsError) {
        dom.feedTopicsList.innerHTML = `<div class="feed-empty">${state.feedTopicsError}</div>`;
        return;
    }
    if (state.feedTopicsCache.length === 0) {
        dom.feedTopicsList.innerHTML = '<div class="feed-empty">No topics</div>';
        return;
    }
    state.feedTopicsCache.forEach((topic) => {
        const topicEl = document.createElement('div');
        topicEl.className = 'feed-topic-item';

        const unreadCount = state.feedTopicsUnread[topic] || 0;
        if (unreadCount > 0) {
            topicEl.classList.add('has-unread');
        }

        topicEl.innerHTML = `<div class="thread-text"></div><div class="dot"></div>`;
        const textEl = topicEl.querySelector('.thread-text');
        textEl.textContent = truncateTopicTitle(topic);
        textEl.title = topic;
        topicEl.dataset.topicId = topic;
        if (topic === state.currentFeedTopicId) {
            topicEl.classList.add('active');
        }
        topicEl.addEventListener('click', async () => {
            if (dom.feedDetailView) dom.feedDetailView.style.display = 'none';
            if (dom.feedArticlesList) dom.feedArticlesList.style.display = 'block';

            if (topic === state.currentFeedTopicId) {
                return;
            }

            state.currentFeedTopicId = topic;
            renderFeedTopics();
            await loadFeedArticles(topic);
        });
        dom.feedTopicsList.appendChild(topicEl);
    });
}

function truncateTopicTitle(text, maxChars = 12) {
    if (!text || text.length <= maxChars) return text;
    return `${text.slice(0, maxChars)}..`;
}

export function renderFeedArticles() {
    if (!dom.feedArticlesList) return;
    if (state.feedArticlesLoading) {
        dom.feedArticlesList.classList.add('is-empty');
        dom.feedArticlesList.innerHTML = '<div class="feed-empty">Loading articles...</div>';
        return;
    }
    if (!state.currentFeedTopicId) {
        dom.feedArticlesList.classList.add('is-empty');
        dom.feedArticlesList.innerHTML = `<div class="feed-empty">
            <div class="chat-empty-title">Select a Topic</div>
            <div class="chat-empty-sub">Articles will be shown here once you select a topic from the left.</div>
        </div>`;
        return;
    }
    if (state.feedArticlesError) {
        dom.feedArticlesList.classList.add('is-empty');
        dom.feedArticlesList.innerHTML = `<div class="feed-empty">${state.feedArticlesError}</div>`;
        return;
    }
    if (state.feedArticlesCache.length === 0) {
        dom.feedArticlesList.classList.add('is-empty');
        dom.feedArticlesList.innerHTML = '<div class="feed-empty">No articles for this topic.</div>';
        return;
    }

    dom.feedArticlesList.classList.remove('is-empty');
    dom.feedArticlesList.innerHTML = '';
    state.feedArticlesCache.forEach((article) => {
        const articleEl = document.createElement('div');
        articleEl.className = 'feed-article-item';
        if (!article.is_read) {
            articleEl.classList.add('has-unread');
        }

        const headerEl = document.createElement('div');
        headerEl.className = 'feed-article-header';
        const unreadDot = document.createElement('span');
        unreadDot.className = 'dot';
        const titleEl = document.createElement('h2');
        titleEl.className = 'feed-article-title';
        titleEl.textContent = article?.title || 'Untitled';
        const timestampEl = document.createElement('span');
        timestampEl.className = 'feed-article-timestamp';
        timestampEl.textContent = formatFeedTimestamp(article?.created_at);
        const contentEl = document.createElement('p');
        contentEl.className = 'feed-article-content';
        contentEl.textContent = article?.content || '';

        headerEl.appendChild(titleEl);
        headerEl.appendChild(timestampEl);
        articleEl.appendChild(headerEl);
        articleEl.appendChild(contentEl);
        articleEl.appendChild(unreadDot);

        articleEl.addEventListener('click', () => {
            // Mark as read if not already
            if (!article.is_read) {
                article.is_read = true;
                articleEl.classList.remove('has-unread');

                // Update unread count
                const topic = article.topic;
                if (topic && state.feedTopicsUnread[topic] > 0) {
                    state.feedTopicsUnread[topic]--;
                    updateTopicDotUI(topic);
                    updateGlobalUnreadDot();
                }

                // Call backend asynchronously
                if (window.go?.app?.App?.MarkFeedItemAsRead && article.topic && article.id) {
                    window.go.app.App.MarkFeedItemAsRead(article.topic, article.id).catch((err) => {
                        console.error('Failed to mark item as read', err);
                    });
                }
            }

            if (dom.feedDetailTitle) dom.feedDetailTitle.textContent = article?.title || 'Untitled';
            if (dom.feedDetailTimestamp) dom.feedDetailTimestamp.textContent = formatFeedTimestamp(article?.created_at);
            renderFeedDetailMarkdown(article?.content || '');
            if (dom.feedArticlesList) dom.feedArticlesList.style.display = 'none';
            if (dom.feedDetailView) dom.feedDetailView.style.display = 'flex';
        });
        dom.feedArticlesList.appendChild(articleEl);
    });
}

export async function loadFeedTopics() {
    if (!window.go?.app?.App?.ListFeedTopicStatuses) {
        state.feedTopicsCache = [];
        state.feedTopicsError = 'Feed is not available';
        renderFeedTopics();
        return;
    }
    if (state.feedTopicsLoading) return;
    state.feedTopicsLoading = true;
    state.feedTopicsError = '';
    renderFeedTopics();
    try {
        await refreshTopicStatuses();

        if (!state.feedTopicsCache.includes(state.currentFeedTopicId)) {
            state.feedTopicsCache.length > 0
                ? (state.currentFeedTopicId = state.feedTopicsCache[0])
                : (state.currentFeedTopicId = '');
        }
    } catch (error) {
        console.error('Failed to load feed topics:', error);
        state.feedTopicsCache = [];
        state.feedTopicsError = 'Failed to load feed topics';
    }
    state.feedTopicsLoading = false;
    renderFeedTopics();
    if (state.feedTopicsError || state.feedTopicsCache.length === 0) {
        state.feedArticlesCache = [];
        state.feedArticlesLoading = false;
        state.feedArticlesError = '';
        renderFeedArticles();
        return;
    }
    await loadFeedArticles(state.currentFeedTopicId);
}

export function updateGlobalUnreadDot() {
    const feedBtn = document.querySelector('.view-btn[data-view="notifications"]');
    const settingsBackBtn = document.querySelector('.settings-header .back-btn');

    if (!feedBtn && !settingsBackBtn) return;

    const totalUnread = Object.values(state.feedTopicsUnread).reduce((a, b) => a + b, 0);
    const hasUnread = totalUnread > 0;

    if (feedBtn) {
        if (hasUnread) {
            feedBtn.classList.add('has-unread');
        } else {
            feedBtn.classList.remove('has-unread');
        }
    }

    if (settingsBackBtn) {
        if (hasUnread) {
            settingsBackBtn.classList.add('has-unread');
        } else {
            settingsBackBtn.classList.remove('has-unread');
        }
    }
}

async function loadFeedArticles(topic) {
    if (!dom.feedArticlesList) return;
    if (!topic) {
        state.feedArticlesCache = [];
        state.feedArticlesLoading = false;
        state.feedArticlesError = '';
        renderFeedArticles();
        return;
    }
    if (!window.go?.app?.App?.LoadFeedTopic) {
        state.feedArticlesCache = [];
        state.feedArticlesLoading = false;
        state.feedArticlesError = 'Feed is not available';
        renderFeedArticles();
        return;
    }
    const requestToken = ++state.feedArticlesRequestToken;
    state.feedArticlesLoading = true;
    state.feedArticlesError = '';
    renderFeedArticles();
    try {
        const items = await window.go.app.App.LoadFeedTopic(topic);
        if (requestToken !== state.feedArticlesRequestToken) return;
        state.feedArticlesCache = Array.isArray(items) ? items : [];
    } catch (error) {
        if (requestToken !== state.feedArticlesRequestToken) return;
        console.error('Failed to load feed articles:', error);
        state.feedArticlesCache = [];
        state.feedArticlesError = 'Failed to load feed articles';
    }
    if (requestToken !== state.feedArticlesRequestToken) return;
    state.feedArticlesLoading = false;
    renderFeedArticles();
}

function renderFeedDetailMarkdown(content) {
    if (!dom.feedDetailContent) return;
    renderMarkdownInto(dom.feedDetailContent, content);
}
