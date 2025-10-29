import { api, isLoggedIn, updateNavigation, getCurrentUserId } from './api.js';

function showToast(message, type = 'success') {
    const toast = document.getElementById('toast');
    toast.textContent = message;
    toast.className = 'toast show' + (type === 'error' ? ' error' : '');
    setTimeout(() => toast.classList.remove('show'), 3000);
}

updateNavigation();

const bookId = new URLSearchParams(window.location.search).get('id');
let selectedRating = 0;
let selectedStatus = 'finished_reading';
let existingRating = null;
let currentSortBy = 'newest';
let currentUserId = getCurrentUserId();

async function loadBook() {
    const loading = document.getElementById('loading');
    const detail = document.getElementById('bookDetail');

    try {
        const book = await api.getBook(bookId);

        loading.classList.add('hidden');
        detail.classList.remove('hidden');

        document.getElementById('bookCover').src = book.cover_url || 'https://via.placeholder.com/400x600?text=No+Cover';
        document.getElementById('bookTitle').textContent = book.title;
        document.getElementById('bookAuthor').textContent = `by ${book.author}`;
        document.getElementById('bookDescription').textContent = book.description || 'No description available.';

        if (book.published_year) {
            document.getElementById('bookYear').textContent = `Published: ${book.published_year}`;
        }
        if (book.isbn) {
            document.getElementById('bookISBN').textContent = `ISBN: ${book.isbn}`;
        }

        await loadRatings();

        if (isLoggedIn()) {
            document.getElementById('ratingSection').classList.remove('hidden');
            await loadExistingRating();
            setupRatingForm();
        } else {
            document.getElementById('loginPrompt').classList.remove('hidden');
        }

    } catch (error) {
        console.error('Error loading book:', error);
        loading.classList.add('hidden');
        loading.innerHTML = '<p class="text-red-400">Failed to load book</p>';
    }
}

async function loadRatings() {
    try {
        const ratings = await api.getRatings(bookId, currentSortBy);

        const avgRating = ratings.average_rating || 0;
        document.getElementById('avgRating').textContent = avgRating.toFixed(1) + '/10';
        document.getElementById('totalRatings').textContent = `${ratings.total_ratings} rating${ratings.total_ratings !== 1 ? 's' : ''}`;

        const reviewsList = document.getElementById('reviewsList');
        const noReviews = document.getElementById('noReviews');

        if (ratings.ratings && ratings.ratings.length > 0) {
            reviewsList.innerHTML = ratings.ratings.map(r => renderReview(r)).join('');
            setupReviewInteractions();
        } else {
            noReviews.classList.remove('hidden');
        }
    } catch (error) {
        console.error('Error loading ratings:', error);
    }
}

function renderReview(r) {
    const ratingValue = r.rating?.rating || r.rating;
    const isOwnReview = currentUserId && r.user_id === currentUserId;

    return `
        <div class="review-card" data-review-id="${r.id}">
            <div class="flex justify-between items-start mb-2">
                <div>
                    <p class="font-bold cursor-pointer hover:text-blue-400" 
                       onclick="window.location.href='user-profile.html?id=${r.user_id}'">
                        ${r.username}
                    </p>
                    ${ratingValue > 0 ? `<div class="rating-badge">${ratingValue}/10</div>` : ''}
                </div>
                <div class="flex items-center gap-3">
                    <p class="text-sm text-gray-500">${new Date(r.created_at).toLocaleDateString()}</p>
                    ${isOwnReview ? `<button class="edit-review-btn text-blue-400 hover:text-blue-300 text-sm" data-review-id="${r.id}" data-rating="${ratingValue}" data-review="${r.review || ''}">Edit</button>` : ''}
                </div>
            </div>
            ${r.review ? `<p class="text-gray-400 mt-2 review-text-${r.id}">${r.review}</p>` : ''}
            
            <!-- Edit form (hidden by default) -->
            <div class="edit-form hidden mt-3" id="edit-form-${r.id}">
                <div class="flex gap-2 mb-2">
                    ${[1,2,3,4,5,6,7,8,9,10].map(num =>
        `<button class="rating-btn edit-rating-btn ${num === ratingValue ? 'selected' : ''}" data-rating="${num}">${num}</button>`
    ).join('')}
                </div>
                <textarea class="w-full bg-gray-700 text-white p-2 rounded" rows="3" id="edit-review-text-${r.id}">${r.review || ''}</textarea>
                <div class="flex gap-2 mt-2">
                    <button class="btn-primary text-sm save-edit-btn" data-review-id="${r.id}">Save</button>
                    <button class="btn-secondary text-sm cancel-edit-btn" data-review-id="${r.id}">Cancel</button>
                </div>
            </div>

            <!-- Like and Comment buttons -->
            <div class="flex items-center gap-4 mt-3 text-sm">
                <button class="like-btn flex items-center gap-1 ${r.liked_by_user ? 'text-red-400' : 'text-gray-400'} hover:text-red-400 transition" 
                        data-review-id="${r.id}" 
                        data-liked="${r.liked_by_user}">
                    <svg class="w-5 h-5 ${r.liked_by_user ? 'fill-current' : ''}" fill="${r.liked_by_user ? 'currentColor' : 'none'}" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z"></path>
                    </svg>
                    <span class="like-count">${r.like_count || 0}</span>
                </button>
                <button class="comment-toggle-btn flex items-center gap-1 text-gray-400 hover:text-blue-400 transition" data-review-id="${r.id}">
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"></path>
                    </svg>
                    <span>Comments (<span class="comment-count">${r.comment_count || 0}</span>)</span>
                </button>
            </div>

            <!-- Comments section (hidden by default) -->
            <div class="comments-section hidden mt-4 border-t border-gray-700 pt-4" id="comments-${r.id}">
                <div class="comments-list space-y-2 mb-3" id="comments-list-${r.id}">
                    <!-- Comments loaded here -->
                </div>
                ${isLoggedIn() ? `
                    <div class="flex gap-2">
                        <input type="text" placeholder="Add a comment..." class="flex-1 bg-gray-700 text-white px-3 py-2 rounded" id="comment-input-${r.id}">
                        <button class="btn-primary text-sm add-comment-btn" data-review-id="${r.id}">Post</button>
                    </div>
                ` : '<p class="text-gray-500 text-sm">Login to comment</p>'}
            </div>
        </div>
    `;
}

function setupReviewInteractions() {
    // Like buttons
    document.querySelectorAll('.like-btn').forEach(btn => {
        btn.addEventListener('click', async (e) => {
            if (!isLoggedIn()) {
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

    // Comment toggle
    document.querySelectorAll('.comment-toggle-btn').forEach(btn => {
        btn.addEventListener('click', async (e) => {
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

    // Add comment
    document.querySelectorAll('.add-comment-btn').forEach(btn => {
        btn.addEventListener('click', async (e) => {
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

    // Edit review
    document.querySelectorAll('.edit-review-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const reviewId = btn.dataset.reviewId;
            document.querySelector(`.review-text-${reviewId}`)?.classList.add('hidden');
            document.getElementById(`edit-form-${reviewId}`).classList.remove('hidden');
            btn.classList.add('hidden');
        });
    });

    // Cancel edit
    document.querySelectorAll('.cancel-edit-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const reviewId = btn.dataset.reviewId;
            document.querySelector(`.review-text-${reviewId}`)?.classList.remove('hidden');
            document.getElementById(`edit-form-${reviewId}`).classList.add('hidden');
            document.querySelector(`.edit-review-btn[data-review-id="${reviewId}"]`).classList.remove('hidden');
        });
    });

    // Save edit
    document.querySelectorAll('.save-edit-btn').forEach(btn => {
        btn.addEventListener('click', async (e) => {
            const reviewId = btn.dataset.reviewId;
            const editForm = document.getElementById(`edit-form-${reviewId}`);
            const selectedRatingBtn = editForm.querySelector('.edit-rating-btn.selected');
            const rating = parseInt(selectedRatingBtn?.dataset.rating || 0);
            const review = document.getElementById(`edit-review-text-${reviewId}`).value;

            if (!rating) {
                showToast('Please select a rating', 'error');
                return;
            }

            try {
                await api.updateRating(reviewId, rating, review);
                showToast('Review updated!');
                setTimeout(() => location.reload(), 1000);
            } catch (error) {
                showToast('Failed to update review', 'error');
            }
        });
    });

    // Edit form rating buttons
    document.querySelectorAll('.edit-rating-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.preventDefault();
            const form = btn.closest('.edit-form');
            form.querySelectorAll('.edit-rating-btn').forEach(b => b.classList.remove('selected'));
            btn.classList.add('selected');
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
                        <p class="text-sm font-bold text-blue-400">${c.username}</p>
                        <p class="text-sm text-gray-300">${c.text}</p>
                    </div>
                    ${currentUserId === c.user_id ? `
                        <button class="delete-comment-btn text-red-400 hover:text-red-300 text-xs" data-comment-id="${c.id}" data-review-id="${reviewId}">Delete</button>
                    ` : ''}
                </div>
            `).join('');

            // Setup delete buttons
            commentsList.querySelectorAll('.delete-comment-btn').forEach(btn => {
                btn.addEventListener('click', async () => {
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

// Sort dropdown
document.getElementById('sortSelect')?.addEventListener('change', async (e) => {
    console.log('Sort changed to:', e.target.value);
    currentSortBy = e.target.value;
    await loadRatings();
});

async function loadExistingRating() {
    try {
        existingRating = await api.getMyRatingForBook(bookId);

        if (existingRating) {
            selectedStatus = existingRating.status;
            const statusBtn = document.querySelector(`.status-btn[data-status="${existingRating.status}"]`);
            if (statusBtn) {
                statusBtn.classList.add('selected');
            }

            if (existingRating.rating > 0) {
                selectedRating = existingRating.rating;
                document.getElementById('ratingValue').value = selectedRating;
                const ratingBtn = document.querySelector(`.rating-btn[data-rating="${existingRating.rating}"]`);
                if (ratingBtn) {
                    ratingBtn.classList.add('selected');
                }
            }

            if (existingRating.review) {
                document.getElementById('reviewText').value = existingRating.review;
            }
        }
    } catch (error) {
        if (!error.message.includes('404')) {
            console.error('Error loading existing rating:', error);
        }
    }
}

function setupRatingForm() {
    const statusBtns = document.querySelectorAll('.status-btn');
    statusBtns.forEach(btn => {
        btn.addEventListener('click', async () => {
            selectedStatus = btn.dataset.status;
            statusBtns.forEach(b => b.classList.remove('selected'));
            btn.classList.add('selected');

            if (selectedStatus === 'to_read' || selectedStatus === 'currently_reading') {
                try {
                    await api.createRating(bookId, 0, '', selectedStatus);
                    showToast(`Marked as ${getStatusLabel(selectedStatus)}!`, 'success');
                    setTimeout(() => location.reload(), 1000);
                } catch (error) {
                    console.error('Status save error:', error);
                    showToast('Failed to save status: ' + error.message, 'error');
                }
            }
        });
    });

    const ratingBtns = document.querySelectorAll('.rating-btn');
    ratingBtns.forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.preventDefault();
            selectedRating = parseInt(btn.dataset.rating);
            document.getElementById('ratingValue').value = selectedRating;

            ratingBtns.forEach(b => b.classList.remove('selected'));
            btn.classList.add('selected');
        });
    });

    document.getElementById('ratingForm').addEventListener('submit', async (e) => {
        e.preventDefault();

        if (!selectedRating) {
            showToast('Please select a rating', 'error');
            return;
        }

        const review = document.getElementById('reviewText').value;

        try {
            await api.createRating(bookId, selectedRating, review, selectedStatus);
            showToast('Rating submitted!', 'success');
            setTimeout(() => location.reload(), 1500);
        } catch (error) {
            console.error('Rating error:', error);
            showToast('Failed to submit rating: ' + error.message, 'error');
        }
    });
}

function getStatusLabel(status) {
    switch(status) {
        case 'to_read': return 'Want to Read';
        case 'currently_reading': return 'Currently Reading';
        case 'finished_reading': return 'Finished';
        default: return status;
    }
}

loadBook();