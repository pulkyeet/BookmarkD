(function () {
    'use strict';

    const API_BASE = 'https://bookmarkd.fly.dev/api';
    const SITE_BASE = 'https://bookmarkd.fly.dev';

    const containers = document.querySelectorAll('[data-bookmarkd-embed]');

    containers.forEach(container => {
        const type = container.dataset.bookmarkdEmbed;
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
            container.innerHTML = '<p style="color: #f87171; font-size: 0.875rem; font-family: sans-serif;">Failed to load books</p>';
        }
    }

    async function loadListEmbed(container, listId, count, style) {
        try {
            const res = await fetch(`${API_BASE}/embed/lists/${listId}?count=${count}`);
            if (!res.ok) throw new Error('Failed to load');
            const data = await res.json();
            renderListEmbed(container, data, style);
        } catch (err) {
            container.innerHTML = '<p style="color: #f87171; font-size: 0.875rem; font-family: sans-serif;">Failed to load list</p>';
        }
    }

    function renderUserEmbed(container, data, style) {
        const css = getStyles();

        let gridClass = 'bookmarkd-grid';
        if (style === 'list') gridClass = 'bookmarkd-list';
        if (style === 'minimal') gridClass = 'bookmarkd-minimal';

        const booksHTML = data.books.map(book => {
            if (style === 'list') {
                return `
                    <a href="${SITE_BASE}/book-detail.html?id=${book.book_id}" target="_blank" class="bookmarkd-card bookmarkd-card-list">
                        <img src="${book.cover_url || ''}" alt="${book.title}">
                        <div class="bookmarkd-card-content">
                            <h3>${book.title}</h3>
                            <p class="bookmarkd-author">${book.author}</p>
                            ${book.rating > 0 ? `<span class="bookmarkd-rating">${book.rating}/10</span>` : ''}
                            ${book.review_snippet ? `<p class="bookmarkd-review">${book.review_snippet}</p>` : ''}
                        </div>
                    </a>`;
            } else if (style === 'minimal') {
                return `
                    <a href="${SITE_BASE}/book-detail.html?id=${book.book_id}" target="_blank" class="bookmarkd-card bookmarkd-card-minimal">
                        <img src="${book.cover_url || ''}" alt="${book.title}">
                    </a>`;
            } else {
                return `
                    <a href="${SITE_BASE}/book-detail.html?id=${book.book_id}" target="_blank" class="bookmarkd-card">
                        <img src="${book.cover_url || ''}" alt="${book.title}">
                        <h3>${book.title}</h3>
                        <p class="bookmarkd-author">${book.author}</p>
                        ${book.rating > 0 ? `<span class="bookmarkd-rating">${book.rating}/10</span>` : ''}
                    </a>`;
            }
        }).join('');

        container.innerHTML = `
            ${css}
            <div class="bookmarkd-embed">
                <div class="bookmarkd-header">
                    <h2>${data.username}'s Top Books</h2>
                    <div class="bookmarkd-credit">Curated on <a href="${SITE_BASE}" target="_blank">BookmarkD</a></div>
                </div>
                <div class="${gridClass}">${booksHTML}</div>
                <div class="bookmarkd-footer">
                    <a href="${SITE_BASE}/user-profile.html?id=${data.user_id}" target="_blank" class="bookmarkd-badge">
                        View full profile &rarr;
                    </a>
                </div>
            </div>`;
    }

    function renderListEmbed(container, data, style) {
        const css = getStyles();

        let gridClass = 'bookmarkd-grid';
        if (style === 'list') gridClass = 'bookmarkd-list';
        if (style === 'minimal') gridClass = 'bookmarkd-minimal';

        const booksHTML = data.books.map((book, idx) => {
            if (style === 'list') {
                return `
                    <a href="${SITE_BASE}/book-detail.html?id=${book.book_id}" target="_blank" class="bookmarkd-card bookmarkd-card-list">
                        <div class="bookmarkd-number">${idx + 1}</div>
                        <img src="${book.cover_url || ''}" alt="${book.title}">
                        <div class="bookmarkd-card-content">
                            <h3>${book.title}</h3>
                            <p class="bookmarkd-author">${book.author}</p>
                        </div>
                    </a>`;
            } else if (style === 'minimal') {
                return `
                    <a href="${SITE_BASE}/book-detail.html?id=${book.book_id}" target="_blank" class="bookmarkd-card bookmarkd-card-minimal">
                        <img src="${book.cover_url || ''}" alt="${book.title}">
                    </a>`;
            } else {
                return `
                    <a href="${SITE_BASE}/book-detail.html?id=${book.book_id}" target="_blank" class="bookmarkd-card">
                        <img src="${book.cover_url || ''}" alt="${book.title}">
                        <h3>${book.title}</h3>
                        <p class="bookmarkd-author">${book.author}</p>
                    </a>`;
            }
        }).join('');

        container.innerHTML = `
            ${css}
            <div class="bookmarkd-embed">
                <div class="bookmarkd-header">
                    <h2>${data.list_name}</h2>
                    ${data.description ? `<p class="bookmarkd-description">${data.description}</p>` : ''}
                    <div class="bookmarkd-credit">by ${data.username} &middot; Curated on <a href="${SITE_BASE}" target="_blank">BookmarkD</a></div>
                </div>
                <div class="${gridClass}">${booksHTML}</div>
                <div class="bookmarkd-footer">
                    <a href="${SITE_BASE}/list-detail.html?id=${data.list_id}" target="_blank" class="bookmarkd-badge">
                        View full list &rarr;
                    </a>
                </div>
            </div>`;
    }

    function getStyles() {
        return `
            <style>
                @import url('https://fonts.googleapis.com/css2?family=DM+Sans:wght@400;500;600&family=Playfair+Display:wght@600;700&display=swap');

                .bookmarkd-embed {
                    font-family: 'DM Sans', -apple-system, BlinkMacSystemFont, sans-serif;
                    max-width: 100%;
                    margin: 0 auto;
                    background: #09090b;
                    color: #f5f5f4;
                    padding: 20px;
                    border-radius: 14px;
                    border: 1px solid #25252b;
                    -webkit-font-smoothing: antialiased;
                    position: relative;
                }

                .bookmarkd-header {
                    margin-bottom: 1.25rem;
                    padding-bottom: 1rem;
                    border-bottom: 1px solid #25252b;
                }

                .bookmarkd-header h2 {
                    font-family: 'Playfair Display', Georgia, serif;
                    font-size: 1.375rem;
                    font-weight: 700;
                    margin: 0 0 0.25rem 0;
                    letter-spacing: -0.01em;
                    color: #f5f5f4;
                }

                .bookmarkd-description {
                    font-size: 0.8125rem;
                    color: #a8a8b3;
                    margin: 0.25rem 0 0.5rem;
                    line-height: 1.5;
                }

                .bookmarkd-credit {
                    font-size: 0.6875rem;
                    color: #63636e;
                    letter-spacing: 0.04em;
                    text-transform: uppercase;
                }

                .bookmarkd-credit a {
                    color: #d4a574;
                    text-decoration: none;
                    transition: color 0.2s;
                }

                .bookmarkd-credit a:hover { color: #e2bb91; }

                .bookmarkd-grid {
                    display: grid;
                    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
                    gap: 14px;
                }

                .bookmarkd-list {
                    display: grid;
                    grid-template-columns: 1fr;
                    gap: 10px;
                }

                .bookmarkd-minimal {
                    display: grid;
                    grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
                    gap: 10px;
                }

                .bookmarkd-card {
                    background: #131316;
                    border: 1px solid #25252b;
                    border-radius: 10px;
                    padding: 10px;
                    transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
                    cursor: pointer;
                    text-decoration: none;
                    color: inherit;
                    display: block;
                }

                .bookmarkd-card:hover {
                    border-color: rgba(212, 165, 116, 0.3);
                    transform: translateY(-3px);
                    box-shadow: 0 12px 32px rgba(0, 0, 0, 0.4), 0 0 20px rgba(212, 165, 116, 0.06);
                }

                .bookmarkd-card img {
                    width: 100%;
                    aspect-ratio: 2/3;
                    object-fit: cover;
                    border-radius: 6px;
                    background: linear-gradient(135deg, #1a1a1f, #25252b);
                    margin-bottom: 8px;
                    display: block;
                }

                .bookmarkd-card h3 {
                    font-size: 0.8125rem;
                    font-weight: 600;
                    margin: 0 0 2px 0;
                    color: #f5f5f4;
                    overflow: hidden;
                    text-overflow: ellipsis;
                    white-space: nowrap;
                }

                .bookmarkd-author {
                    font-size: 0.6875rem;
                    color: #63636e;
                    margin: 0;
                    overflow: hidden;
                    text-overflow: ellipsis;
                    white-space: nowrap;
                }

                .bookmarkd-rating {
                    display: inline-flex;
                    align-items: center;
                    margin-top: 6px;
                    background: #d4a574;
                    color: #09090b;
                    font-size: 0.6875rem;
                    font-weight: 700;
                    padding: 2px 7px;
                    border-radius: 4px;
                    letter-spacing: 0.02em;
                }

                .bookmarkd-card-list {
                    display: flex;
                    gap: 12px;
                    padding: 12px;
                    align-items: center;
                }

                .bookmarkd-card-list img {
                    width: 52px;
                    height: 78px;
                    flex-shrink: 0;
                    margin-bottom: 0;
                    border-radius: 4px;
                }

                .bookmarkd-card-content {
                    flex: 1;
                    min-width: 0;
                }

                .bookmarkd-card-content h3 { font-size: 0.875rem; }

                .bookmarkd-review {
                    font-size: 0.75rem;
                    color: #a8a8b3;
                    margin-top: 4px;
                    line-height: 1.5;
                    display: -webkit-box;
                    -webkit-line-clamp: 2;
                    -webkit-box-orient: vertical;
                    overflow: hidden;
                }

                .bookmarkd-number {
                    font-family: 'Playfair Display', Georgia, serif;
                    font-size: 1.25rem;
                    font-weight: 700;
                    color: #25252b;
                    width: 2rem;
                    flex-shrink: 0;
                    text-align: center;
                }

                .bookmarkd-card-minimal {
                    padding: 0;
                    background: transparent;
                    border: none;
                    border-radius: 8px;
                    overflow: hidden;
                }

                .bookmarkd-card-minimal:hover {
                    transform: scale(1.04);
                    box-shadow: 0 12px 32px rgba(0, 0, 0, 0.5);
                    border: none;
                }

                .bookmarkd-card-minimal img {
                    margin-bottom: 0;
                    border-radius: 8px;
                }

                .bookmarkd-footer {
                    margin-top: 1.25rem;
                    padding-top: 1rem;
                    border-top: 1px solid #25252b;
                    text-align: center;
                }

                .bookmarkd-badge {
                    display: inline-flex;
                    align-items: center;
                    gap: 6px;
                    padding: 0.4rem 1rem;
                    font-size: 0.6875rem;
                    font-weight: 600;
                    border-radius: 6px;
                    background: rgba(212, 165, 116, 0.08);
                    color: #d4a574;
                    text-decoration: none;
                    transition: all 0.25s;
                    letter-spacing: 0.02em;
                }

                .bookmarkd-badge:hover {
                    background: rgba(212, 165, 116, 0.15);
                    color: #e2bb91;
                }
            </style>`;
    }
})();
