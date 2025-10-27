import { api, isLoggedIn, updateNavigation } from './api.js';

updateNavigation();

let currentFeedType = 'all';
let currentOffset = 0;
const LIMIT = 20;
const loggedIn = isLoggedIn();

// Show toggle only if logged in
if (loggedIn) {
    document.getElementById('feedToggle').classList.remove('hidden');
}

// Feed toggle buttons
document.getElementById('allFeedBtn').addEventListener('click', () => {
    currentFeedType = 'all';
    document.getElementById('allFeedBtn').classList.remove('btn-secondary');
    document.getElementById('allFeedBtn').classList.add('btn-primary');
    document.getElementById('followingFeedBtn').classList.add('btn-secondary');
    document.getElementById('followingFeedBtn').classList.remove('btn-primary');
    currentOffset = 0;
    loadFeed(true);
});

document.getElementById('followingFeedBtn').addEventListener('click', () => {
    currentFeedType = 'following';
    document.getElementById('followingFeedBtn').classList.remove('btn-secondary');
    document.getElementById('followingFeedBtn').classList.add('btn-primary');
    document.getElementById('allFeedBtn').classList.add('btn-secondary');
    document.getElementById('allFeedBtn').classList.remove('btn-primary');
    currentOffset = 0;
    loadFeed(true);
});

// Load More button
document.getElementById('loadMoreBtn').addEventListener('click', () => {
    currentOffset += LIMIT;
    loadFeed(false);
});

async function loadFeed(reset = false) {
    const loading = document.getElementById('loading');
    const feedList = document.getElementById('feedList');
    const noActivity = document.getElementById('noActivity');
    const loadMoreContainer = document.getElementById('loadMoreContainer');

    if (reset) {
        loading.classList.remove('hidden');
        feedList.classList.add('hidden');
        noActivity.classList.add('hidden');
        loadMoreContainer.classList.add('hidden');
    }

    try {
        const feed = await api.getFeed(currentFeedType, LIMIT, currentOffset);

        loading.classList.add('hidden');

        if (feed && feed.length > 0) {
            feedList.classList.remove('hidden');

            const feedHTML = feed.map(item => `
                <div class="auth-card hover:border-blue-500 transition cursor-pointer"
                     onclick="window.location.href='book-detail.html?id=${item.book_id}'">
                    <div class="flex gap-4">
                        <img src="${item.book_cover || 'https://via.placeholder.com/100x150'}" 
                             alt="${item.book_title}" 
                             class="w-20 h-30 object-cover rounded">
                        <div class="flex-1">
                            <div class="flex justify-between items-start mb-2">
                                <div>
                                    <p class="text-sm text-gray-400">
                                        <span class="font-bold text-blue-400 hover:underline" 
                                              onclick="event.stopPropagation(); window.location.href='user-profile.html?id=${item.user_id}'">
                                            ${item.username}
                                        </span>
                                        ${getStatusText(item.status)}
                                    </p>
                                    <h3 class="font-bold text-lg mt-1">${item.book_title}</h3>
                                    <p class="text-gray-400 text-sm">${item.book_author}</p>
                                </div>
                                ${item.rating > 0 ? `<div class="rating-badge">${item.rating}/10</div>` : ''}
                            </div>
                            ${item.review ? `<p class="text-gray-400 text-sm mt-2">${item.review}</p>` : ''}
                            <p class="text-xs text-gray-500 mt-2">${formatDate(item.created_at)}</p>
                        </div>
                    </div>
                </div>
            `).join('');

            if (reset) {
                feedList.innerHTML = feedHTML;
            } else {
                feedList.innerHTML += feedHTML;
            }

            // Show/hide Load More button
            if (feed.length === LIMIT) {
                loadMoreContainer.classList.remove('hidden');
            } else {
                loadMoreContainer.classList.add('hidden');
            }
        } else {
            if (reset) {
                noActivity.classList.remove('hidden');
            } else {
                loadMoreContainer.classList.add('hidden');
            }
        }
    } catch (error) {
        console.error('Error loading feed:', error);
        loading.classList.add('hidden');
        feedList.innerHTML = `<p class="text-red-400">Failed to load feed: ${error.message}</p>`;
        feedList.classList.remove('hidden');
    }
}

function getStatusText(status) {
    switch(status) {
        case 'to_read': return 'wants to read';
        case 'currently_reading': return 'is reading';
        case 'finished_reading': return 'finished reading';
        default: return 'rated';
    }
}

function formatDate(dateString) {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now - date;
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return 'just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;
    return date.toLocaleDateString();
}

loadFeed(true);