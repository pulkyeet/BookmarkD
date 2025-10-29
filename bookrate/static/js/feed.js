import { api, isLoggedIn, updateNavigation, getCurrentUserId } from './api.js';

updateNavigation();

let currentFeedType = 'all';
let currentOffset = 0;
const LIMIT = 20;
const loggedIn = isLoggedIn();
const currentUserId = getCurrentUserId();

function showToast(message, type = 'success') {
    const toast = document.getElementById('toast');
    if (!toast) return;
    toast.textContent = message;
    toast.className = 'toast show' + (type === 'error' ? ' error' : '');
    setTimeout(() => toast.classList.remove('show'), 3000);
}

if (loggedIn) {
    document.getElementById('feedToggle').classList.remove('hidden');
}

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

            const feedHTML = feed.map(item => renderFeedItem(item)).join('');

            if (reset) {
                feedList.innerHTML = feedHTML;
            } else {
                feedList.innerHTML += feedHTML;
            }

            setupFeedInteractions();

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

function renderFeedItem(item) {
    const hasReview = item.review && item.review.trim().length > 0;

    return `
        <div class="auth-card" data-item-id="${item.id}">
            <div class="flex gap-4">
                <img src="${item.book_cover || 'https://via.placeholder.com/100x150'}" 
                     alt="${item.book_title}" 
                     class="w-20 h-30 object-cover rounded cursor-pointer hover:opacity-80 transition"
                     onclick="window.location.href='book-detail.html?id=${item.book_id}'">
                <div class="flex-1">
                    <div class="flex justify-between items-start mb-2">
                        <div>
                            <p class="text-sm text-gray-400">
                                <span class="font-bold text-blue-400 hover:underline cursor-pointer" 
                                      onclick="window.location.href='user-profile.html?id=${item.user_id}'">
                                    ${item.username}
                                </span>
                                ${getStatusText(item.status)}
                            </p>
                            <h3 class="font-bold text-lg mt-1 cursor-pointer hover:text-blue-400 transition"
                                onclick="window.location.href='book-detail.html?id=${item.book_id}'">${item.book_title}</h3>
                            <p class="text-gray-400 text-sm">${item.book_author}</p>
                        </div>
                        ${item.rating > 0 ? `<div class="rating-badge">${item.rating}/10</div>` : ''}
                    </div>
                    
                    ${hasReview ? `<p class="text-gray-400 text-sm mt-2">${item.review}</p>` : ''}
                    
                    ${hasReview ? `
                    <!-- Like and Comment buttons (only for reviews) -->
                    <div class="flex items-center gap-4 mt-3 text-sm">
                        <button class="like-btn flex items-center gap-1 ${item.liked_by_user ? 'text-red-400' : 'text-gray-400'} hover:text-red-400 transition" 
                                data-review-id="${item.id}" 
                                data-liked="${item.liked_by_user}">
                            <svg class="w-5 h-5 ${item.liked_by_user ? 'fill-current' : ''}" fill="${item.liked_by_user ? 'currentColor' : 'none'}" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z"></path>
                            </svg>
                            <span class="like-count">${item.like_count || 0}</span>
                        </button>
                        <button class="comment-toggle-btn flex items-center gap-1 text-gray-400 hover:text-blue-400 transition" data-review-id="${item.id}">
                            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"></path>
                            </svg>
                            <span>Comments (<span class="comment-count">${item.comment_count || 0}</span>)</span>
                        </button>
                        <p class="text-xs text-gray-500 ml-auto">${formatDate(item.created_at)}</p>
                    </div>

                    <!-- Comments section -->
                    <div class="comments-section hidden mt-4 border-t border-gray-700 pt-4" id="comments-${item.id}">
                        <div class="comments-list space-y-2 mb-3" id="comments-list-${item.id}">
                        </div>
                        ${loggedIn ? `
                            <div class="flex gap-2">
                                <input type="text" placeholder="Add a comment..." class="flex-1 bg-gray-700 text-white px-3 py-2 rounded" id="comment-input-${item.id}">
                                <button class="btn-primary text-sm add-comment-btn" data-review-id="${item.id}">Post</button>
                            </div>
                        ` : '<p class="text-gray-500 text-sm">Login to comment</p>'}
                    </div>
                    ` : `
                    <!-- Just timestamp for status-only updates -->
                    <p class="text-xs text-gray-500 mt-2">${formatDate(item.created_at)}</p>
                    `}
                </div>
            </div>
        </div>
    `;
}

function setupFeedInteractions() {
    document.querySelectorAll('.like-btn').forEach(btn => {
        if (btn.dataset.hasListener) return;
        btn.dataset.hasListener = 'true';

        btn.addEventListener('click', async (e) => {
            e.stopPropagation();

            if (!loggedIn) {
                showToast('Please login to like reviews', 'error');
                return;
            }

            const reviewId = btn.dataset.reviewId;
            const isLiked = btn.dataset.liked === 'true';

            try {
                if (isLiked) {
                    await api.unlikeRating(reviewId);
                    btn.dataset.liked = 'false';
                    btn.classList.remove('text-red-400');
                    btn.classList.add('text-gray-400');
                    btn.querySelector('svg').classList.remove('fill-current');
                    btn.querySelector('svg').setAttribute('fill', 'none');
                } else {
                    await api.likeRating(reviewId);
                    btn.dataset.liked = 'true';
                    btn.classList.remove('text-gray-400');
                    btn.classList.add('text-red-400');
                    btn.querySelector('svg').classList.add('fill-current');
                    btn.querySelector('svg').setAttribute('fill', 'currentColor');
                }

                const countSpan = btn.querySelector('.like-count');
                const currentCount = parseInt(countSpan.textContent);
                countSpan.textContent = isLiked ? currentCount - 1 : currentCount + 1;

            } catch (error) {
                showToast('Failed to update like', 'error');
            }
        });
    });

    document.querySelectorAll('.comment-toggle-btn').forEach(btn => {
        if (btn.dataset.hasListener) return;
        btn.dataset.hasListener = 'true';

        btn.addEventListener('click', async (e) => {
            e.stopPropagation();
            const reviewId = btn.dataset.reviewId;
            const commentsSection = document.getElementById(`comments-${reviewId}`);

            if (commentsSection.classList.contains('hidden')) {
                commentsSection.classList.remove('hidden');
                await loadComments(reviewId);
            } else {
                commentsSection.classList.add('hidden');
            }
        });
    });

    document.querySelectorAll('.add-comment-btn').forEach(btn => {
        if (btn.dataset.hasListener) return;
        btn.dataset.hasListener = 'true';

        btn.addEventListener('click', async (e) => {
            e.stopPropagation();
            const reviewId = btn.dataset.reviewId;
            const input = document.getElementById(`comment-input-${reviewId}`);
            const text = input.value.trim();

            if (!text) return;

            try {
                await api.createComment(reviewId, text);
                input.value = '';
                await loadComments(reviewId);
                showToast('Comment added!');
            } catch (error) {
                showToast('Failed to add comment', 'error');
            }
        });
    });
}

async function loadComments(reviewId) {
    const commentsList = document.getElementById(`comments-list-${reviewId}`);
    const countSpan = document.querySelector(`.comment-toggle-btn[data-review-id="${reviewId}"] .comment-count`);

    try {
        const comments = await api.getComments(reviewId);

        countSpan.textContent = comments.length;

        if (comments.length > 0) {
            commentsList.innerHTML = comments.map(c => `
                <div class="flex justify-between items-start bg-gray-700/30 p-2 rounded">
                    <div class="flex-1">
                        <p class="text-sm font-bold text-blue-400 cursor-pointer hover:underline" onclick="window.location.href='user-profile.html?id=${c.user_id}'">${c.username}</p>
                        <p class="text-sm text-gray-300">${c.text}</p>
                    </div>
                    ${currentUserId === c.user_id ? `
                        <button class="delete-comment-btn text-red-400 hover:text-red-300 text-xs" data-comment-id="${c.id}" data-review-id="${reviewId}">Delete</button>
                    ` : ''}
                </div>
            `).join('');

            commentsList.querySelectorAll('.delete-comment-btn').forEach(btn => {
                btn.addEventListener('click', async (e) => {
                    e.stopPropagation();
                    try {
                        await api.deleteComment(btn.dataset.commentId);
                        await loadComments(reviewId);
                        showToast('Comment deleted');
                    } catch (error) {
                        showToast('Failed to delete comment', 'error');
                    }
                });
            });
        } else {
            commentsList.innerHTML = '<p class="text-gray-500 text-sm">No comments yet</p>';
        }
    } catch (error) {
        console.error('Error loading comments:', error);
        commentsList.innerHTML = '<p class="text-red-400 text-sm">Failed to load comments</p>';
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