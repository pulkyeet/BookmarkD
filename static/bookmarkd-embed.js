(function() {
    'use strict';

    const API_BASE = 'http://localhost:8080/api';
    const SITE_BASE = 'http://localhost:8080';

    // Find all embed containers
    const containers = document.querySelectorAll('[data-bookmarkd-embed]');

    containers.forEach(container => {
        const type = container.dataset.bookmarkdEmbed; // 'user' or 'list'
        const id = container.dataset.bookmarkdId;
        const count = container.dataset.bookmarkdCount || 5;
        const style = container.dataset.bookmarkdStyle || 'grid';

        if (!id) {
            console.error('BookmarkD: Missing data-bookmarkd-id attribute');
            return;
        }

        if (type === 'user') {
            loadUserEmbed(container, id, count, style);
        } else if (type === 'list') {
            loadListEmbed(container, id, count, style);
        } else {
            console.error('BookmarkD: Invalid embed type. Use "user" or "list"');
        }
    });

    async function loadUserEmbed(container, userId, count, style) {
        try {
            const res = await fetch(`${API_BASE}/embed/users/${userId}/books?count=${count}`);
            if (!res.ok) throw new Error('Failed to load');

            const data = await res.json();
            renderUserEmbed(container, data, style);
        } catch (err) {
            container.innerHTML = '<p style="color: #ef4444;">Failed to load books</p>';
        }
    }

    async function loadListEmbed(container, listId, count, style) {
        try {
            const res = await fetch(`${API_BASE}/embed/lists/${listId}?count=${count}`);
            if (!res.ok) throw new Error('Failed to load');

            const data = await res.json();
            renderListEmbed(container, data, style);
        } catch (err) {
            container.innerHTML = '<p style="color: #ef4444;">Failed to load list</p>';
        }
    }

    function renderUserEmbed(container, data, style) {
        const styles = getStyles();

        let gridClass = 'bookmarkd-grid';
        if (style === 'list') gridClass = 'bookmarkd-list';
        if (style === 'minimal') gridClass = 'bookmarkd-minimal';

        const booksHTML = data.books.map(book => {
            const stars = '★'.repeat(book.rating) + '☆'.repeat(10 - book.rating);

            if (style === 'list') {
                return `
                    <div class="bookmarkd-card bookmarkd-card-list">
                        <img src="${book.cover_url || 'https://via.placeholder.com/80x120/334155/fff'}" alt="${book.title}">
                        <div class="bookmarkd-card-content">
                            <h3>${book.title}</h3>
                            <p class="bookmarkd-author">${book.author}</p>
                            <div class="bookmarkd-stars">${stars}</div>
                            ${book.review_snippet ? `<p class="bookmarkd-review">${book.review_snippet}</p>` : ''}
                        </div>
                    </div>
                `;
            } else if (style === 'minimal') {
                return `
                    <div class="bookmarkd-card bookmarkd-card-minimal">
                        <img src="${book.cover_url || 'https://via.placeholder.com/200x300/334155/fff'}" alt="${book.title}">
                    </div>
                `;
            } else {
                return `
                    <div class="bookmarkd-card">
                        <img src="${book.cover_url || 'https://via.placeholder.com/200x300/334155/fff'}" alt="${book.title}">
                        <h3>${book.title}</h3>
                        <p class="bookmarkd-author">${book.author}</p>
                        <div class="bookmarkd-stars">${stars}</div>
                    </div>
                `;
            }
        }).join('');

        container.innerHTML = `
            ${styles}
            <div class="bookmarkd-embed">
                <div class="bookmarkd-header">
                    <h2>${data.username}'s Top Books</h2>
                    <div class="bookmarkd-credit">Powered by BookmarkD</div>
                </div>
                <div class="${gridClass}">
                    ${booksHTML}
                </div>
                <div class="bookmarkd-footer">
                    <a href="${SITE_BASE}/user-profile.html?id=${data.user_id}" target="_blank" class="bookmarkd-badge">
                        View full profile on BookmarkD →
                    </a>
                </div>
            </div>
        `;
    }

    function renderListEmbed(container, data, style) {
        const styles = getStyles();

        let gridClass = 'bookmarkd-grid';
        if (style === 'list') gridClass = 'bookmarkd-list';
        if (style === 'minimal') gridClass = 'bookmarkd-minimal';

        const booksHTML = data.books.map((book, idx) => {
            if (style === 'list') {
                return `
                    <div class="bookmarkd-card bookmarkd-card-list">
                        <div class="bookmarkd-number">${idx + 1}</div>
                        <img src="${book.cover_url || 'https://via.placeholder.com/80x120/334155/fff'}" alt="${book.title}">
                        <div class="bookmarkd-card-content">
                            <h3>${book.title}</h3>
                            <p class="bookmarkd-author">${book.author}</p>
                        </div>
                    </div>
                `;
            } else if (style === 'minimal') {
                return `
                    <div class="bookmarkd-card bookmarkd-card-minimal">
                        <img src="${book.cover_url || 'https://via.placeholder.com/200x300/334155/fff'}" alt="${book.title}">
                    </div>
                `;
            } else {
                return `
                    <div class="bookmarkd-card">
                        <img src="${book.cover_url || 'https://via.placeholder.com/200x300/334155/fff'}" alt="${book.title}">
                        <h3>${book.title}</h3>
                        <p class="bookmarkd-author">${book.author}</p>
                    </div>
                `;
            }
        }).join('');

        container.innerHTML = `
            ${styles}
            <div class="bookmarkd-embed">
                <div class="bookmarkd-header">
                    <h2>${data.list_name}</h2>
                    ${data.description ? `<p class="bookmarkd-description">${data.description}</p>` : ''}
                    <div class="bookmarkd-credit">by ${data.username} • Powered by BookmarkD</div>
                </div>
                <div class="${gridClass}">
                    ${booksHTML}
                </div>
                <div class="bookmarkd-footer">
                    <a href="${SITE_BASE}/list-detail.html?id=${data.list_id}" target="_blank" class="bookmarkd-badge">
                        View full list on BookmarkD →
                    </a>
                </div>
            </div>
        `;
    }

    function getStyles() {
        return `
            <style>
                .bookmarkd-embed {
                    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
                    max-width: 100%;
                    margin: 0 auto;
                }
                .bookmarkd-header {
                    margin-bottom: 1.5rem;
                }
                .bookmarkd-header h2 {
                    font-size: 1.5rem;
                    font-weight: bold;
                    margin: 0 0 0.5rem 0;
                    color: #f9fafb;
                }
                .bookmarkd-description {
                    font-size: 0.875rem;
                    color: #9ca3af;
                    margin: 0.5rem 0;
                }
                .bookmarkd-credit {
                    font-size: 0.75rem;
                    color: #6b7280;
                }
                .bookmarkd-grid {
                    display: grid;
                    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
                    gap: 1rem;
                }
                .bookmarkd-list {
                    display: grid;
                    grid-template-columns: 1fr;
                    gap: 1rem;
                }
                .bookmarkd-minimal {
                    display: grid;
                    grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
                    gap: 0.75rem;
                }
                .bookmarkd-card {
                    background: #1e293b;
                    border: 1px solid #334155;
                    border-radius: 8px;
                    padding: 0.75rem;
                    transition: all 0.2s;
                }
                .bookmarkd-card:hover {
                    border-color: #3b82f6;
                    transform: translateY(-2px);
                }
                .bookmarkd-card img {
                    width: 100%;
                    aspect-ratio: 2/3;
                    object-fit: cover;
                    border-radius: 4px;
                    background: #334155;
                    margin-bottom: 0.5rem;
                }
                .bookmarkd-card h3 {
                    font-size: 0.875rem;
                    font-weight: 600;
                    margin: 0 0 0.25rem 0;
                    color: #f9fafb;
                    overflow: hidden;
                    text-overflow: ellipsis;
                    white-space: nowrap;
                }
                .bookmarkd-author {
                    font-size: 0.75rem;
                    color: #9ca3af;
                    margin: 0;
                    overflow: hidden;
                    text-overflow: ellipsis;
                    white-space: nowrap;
                }
                .bookmarkd-stars {
                    font-size: 0.75rem;
                    color: #fbbf24;
                    margin-top: 0.25rem;
                }
                .bookmarkd-card-list {
                    display: flex;
                    gap: 0.75rem;
                    padding: 1rem;
                }
                .bookmarkd-card-list img {
                    width: 64px;
                    height: 96px;
                    flex-shrink: 0;
                    margin-bottom: 0;
                }
                .bookmarkd-card-content {
                    flex: 1;
                    min-width: 0;
                }
                .bookmarkd-review {
                    font-size: 0.75rem;
                    color: #d1d5db;
                    margin-top: 0.5rem;
                    line-height: 1.4;
                }
                .bookmarkd-number {
                    font-size: 1.5rem;
                    font-weight: bold;
                    color: #4b5563;
                    width: 2rem;
                    flex-shrink: 0;
                }
                .bookmarkd-card-minimal {
                    padding: 0;
                    background: transparent;
                    border: none;
                }
                .bookmarkd-card-minimal:hover {
                    transform: scale(1.05);
                }
                .bookmarkd-card-minimal img {
                    margin-bottom: 0;
                }
                .bookmarkd-footer {
                    margin-top: 1.5rem;
                    text-align: center;
                }
                .bookmarkd-badge {
                    display: inline-block;
                    padding: 0.5rem 1rem;
                    font-size: 0.75rem;
                    border-radius: 6px;
                    background: rgba(59, 130, 246, 0.1);
                    color: #60a5fa;
                    text-decoration: none;
                    transition: all 0.2s;
                }
                .bookmarkd-badge:hover {
                    background: rgba(59, 130, 246, 0.2);
                    color: #3b82f6;
                }
            </style>
        `;
    }
})();