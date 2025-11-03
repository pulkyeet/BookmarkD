import { api, isLoggedIn, getCurrentUserId, updateNavigation } from './api.js';

updateNavigation();

const bookId = parseInt(new URLSearchParams(window.location.search).get('id'));
const currentUserId = getCurrentUserId();
let userLists = [];
let selectedRating = null;
let selectedStatus = null;
let hasExistingRating = false;

if (!bookId) {
    window.location.href = 'books.html';
}

// Show My Lists link if logged in
if (isLoggedIn()) {
    document.getElementById('myListsLink')?.classList.remove('hidden');
}

function showToast(message, isError = false) {
    const toast = document.getElementById('toast');
    toast.textContent = message;
    toast.classList.toggle('error', isError);
    toast.classList.add('show');
    setTimeout(() => toast.classList.remove('show'), 3000);
}

async function loadBookDetails() {
    try {
        const book = await api.getBook(bookId);

        document.getElementById('bookTitle').textContent = book.title;
        document.getElementById('bookAuthor').textContent = `by ${book.author}`;
        document.getElementById('bookCover').src = book.cover_url || 'https://via.placeholder.com/400x600?text=No+Cover';

        if (book.description) {
            document.getElementById('bookDescription').textContent = book.description;
        }

        if (book.published_year) {
            document.getElementById('publishedYear').textContent = `Published: ${book.published_year}`;
        }

        if (book.isbn) {
            document.getElementById('bookISBN').textContent = `ISBN: ${book.isbn}`;
        }

        // Display genres
        if (book.genres && book.genres.length > 0) {
            document.getElementById('bookGenres').innerHTML = book.genres
                .map(g => `<span class="genre-badge">${g.name}</span>`)
                .join('');
        }

        document.getElementById('loading').classList.add('hidden');
        document.getElementById('bookDetail').classList.remove('hidden');

        if (isLoggedIn()) {
            document.getElementById('ratingSection').classList.remove('hidden');
            await loadExistingRating();
        } else {
            document.getElementById('loginPrompt').classList.remove('hidden');
        }

    } catch (error) {
        console.error('Error loading book:', error);
        showToast('Failed to load book details', true);
    }
}

async function loadExistingRating() {
    try {
        const rating = await api.getMyRatingForBook(bookId);
        if (rating) {
            hasExistingRating = true;

            // Select the rating button if rating exists
            if (rating.rating > 0) {
                selectedRating = rating.rating;
                const ratingBtn = document.querySelector(`.rating-btn[data-rating="${rating.rating}"]`);
                if (ratingBtn) {
                    ratingBtn.classList.add('selected');
                }
                document.getElementById('ratingValue').value = rating.rating;
            }

            // Select the status button
            selectedStatus = rating.status;
            const statusBtn = document.querySelector(`.status-btn[data-status="${rating.status}"]`);
            if (statusBtn) {
                statusBtn.classList.add('selected');
            }

            // Fill review
            if (rating.review) {
                document.getElementById('reviewText').value = rating.review;
            }

            // Change submit button text
            document.querySelector('#ratingForm button[type="submit"]').textContent = 'Update Rating';
        }
        // If no existing rating, don't select anything - let user choose
    } catch (error) {
        // No existing rating - that's fine, don't select any default
    }
}

async function loadRatings(sortBy = 'newest') {
    try {
        const data = await api.getRatings(bookId, sortBy);

        document.getElementById('avgRating').textContent = data.average_rating.toFixed(1);
        document.getElementById('totalRatings').textContent = `${data.total_ratings} rating${data.total_ratings !== 1 ? 's' : ''}`;

        const container = document.getElementById('reviewsList');
        const noReviews = document.getElementById('noReviews');

        // Filter out ratings without reviews
        const ratingsWithReviews = data.ratings.filter(r => r.review && r.review.trim());

        if (ratingsWithReviews.length === 0) {
            container.classList.add('hidden');
            noReviews.classList.remove('hidden');
            return;
        }

        container.classList.remove('hidden');
        noReviews.classList.add('hidden');

        container.innerHTML = ratingsWithReviews.map(rating => {
            const isOwn = currentUserId && rating.user_id === currentUserId;

            return `
                <div class="bg-gray-800 rounded-lg p-6">
                    <div class="flex justify-between items-start mb-3">
                        <div>
                            <a href="user-profile.html?id=${rating.user_id}" class="font-semibold text-blue-400 hover:text-blue-300">${rating.username}</a>
                            <div class="flex items-center gap-2 mt-1">
                                <span class="rating-badge">${rating.rating}/10</span>
                                <span class="text-gray-500 text-sm">${new Date(rating.created_at).toLocaleDateString()}</span>
                            </div>
                        </div>
                    </div>
                    
                    <p class="text-gray-300 mb-4">${rating.review}</p>
                    
                    <div class="flex items-center gap-4 text-sm">
                        ${!isOwn ? `
                            <button onclick="toggleLike(${rating.id})" 
                                    class="flex items-center gap-1 ${rating.liked_by_user ? 'text-red-500' : 'text-gray-400'} hover:text-red-400">
                                <span id="like-icon-${rating.id}">${rating.liked_by_user ? '❤️' : '🤍'}</span>
                                <span id="like-count-${rating.id}">${rating.like_count}</span>
                            </button>
                        ` : `
                            <span class="flex items-center gap-1 text-gray-600">
                                <span>❤️</span>
                                <span>${rating.like_count}</span>
                            </span>
                        `}
                        <button onclick="toggleComments(${rating.id})" class="text-gray-400 hover:text-blue-400">
                            💬 <span id="comment-count-${rating.id}">${rating.comment_count}</span>
                        </button>
                    </div>
                    
                    <div id="comments-${rating.id}" class="hidden mt-4 pl-4 border-l-2 border-gray-700">
                        <div id="comments-list-${rating.id}"></div>
                        ${isLoggedIn() ? `
                            <div class="mt-4">
                                <textarea id="comment-input-${rating.id}" 
                                          class="input-field w-full" 
                                          rows="2" 
                                          placeholder="Add a comment..."></textarea>
                                <button onclick="addComment(${rating.id})" 
                                        class="btn-primary mt-2">Post Comment</button>
                            </div>
                        ` : ''}
                    </div>
                </div>
            `;
        }).join('');

    } catch (error) {
        console.error('Error loading ratings:', error);
    }
}

async function loadSimilarBooks() {
    try {
        const similarBooks = await api.getSimilarBooks(bookId, 6);

        if (similarBooks.length === 0) {
            return;
        }

        document.getElementById('similarBooksSection').classList.remove('hidden');
        document.getElementById('similarBooksGrid').innerHTML = similarBooks.map(book => `
            <div class="cursor-pointer" onclick="window.location.href='book-detail.html?id=${book.book_id}'">
                <img src="${book.cover_url || 'https://via.placeholder.com/200x300'}" 
                     alt="${book.title}" 
                     class="w-full h-64 object-cover rounded-lg mb-2">
                <h4 class="font-semibold text-sm line-clamp-2">${book.title}</h4>
                <p class="text-gray-400 text-xs">${book.author}</p>
                <p class="text-blue-400 text-sm mt-1">⭐ ${book.avg_rating.toFixed(1)}</p>
            </div>
        `).join('');

    } catch (error) {
        console.error('Error loading similar books:', error);
    }
}

async function loadUserLists() {
    if (!isLoggedIn()) return;

    try {
        userLists = await api.getMyLists();
    } catch (error) {
        console.error('Error loading user lists:', error);
    }
}

// Status button handlers - immediately save when clicked
document.querySelectorAll('.status-btn').forEach(btn => {
    btn.addEventListener('click', async function() {
        document.querySelectorAll('.status-btn').forEach(b => b.classList.remove('selected'));
        this.classList.add('selected');
        selectedStatus = this.dataset.status;

        // Auto-save status change
        try {
            const rating = selectedRating || 0;
            const review = document.getElementById('reviewText').value;
            await api.createRating(bookId, rating, review, selectedStatus);

            const statusText = {
                'to_read': 'Added to Want to Read',
                'currently_reading': 'Added to Currently Reading',
                'finished_reading': 'Marked as Finished'
            };
            showToast(statusText[selectedStatus]);
            loadRatings();
        } catch (error) {
            console.error('Error saving status:', error);
            showToast('Failed to save status', true);
        }
    });
});

// Rating button handlers
document.querySelectorAll('.rating-btn').forEach(btn => {
    btn.addEventListener('click', function() {
        document.querySelectorAll('.rating-btn').forEach(b => b.classList.remove('selected'));
        this.classList.add('selected');
        selectedRating = parseInt(this.dataset.rating);
        document.getElementById('ratingValue').value = selectedRating;
    });
});

// Rating form - for submitting rating + review together
// Rating form - for submitting rating + review together
document.getElementById('ratingForm').addEventListener('submit', async (e) => {
    e.preventDefault();

    if (!isLoggedIn()) {
        showToast('Please log in to rate books', true);
        return;
    }

    if (!selectedRating) {
        showToast('Please select a rating', true);
        return;
    }

    // Default to finished_reading if no status selected
    const status = selectedStatus || 'finished_reading';
    const review = document.getElementById('reviewText').value;

    try {
        await api.createRating(bookId, selectedRating, review, status);
        showToast(hasExistingRating ? 'Rating updated!' : 'Rating submitted!');

        // Update UI to show selected status if it was defaulted
        if (!selectedStatus) {
            selectedStatus = 'finished_reading';
            const finishedBtn = document.querySelector('.status-btn[data-status="finished_reading"]');
            if (finishedBtn) {
                document.querySelectorAll('.status-btn').forEach(b => b.classList.remove('selected'));
                finishedBtn.classList.add('selected');
            }
        }

        loadRatings();
        hasExistingRating = true;
        document.querySelector('#ratingForm button[type="submit"]').textContent = 'Update Rating';
    } catch (error) {
        console.error('Error submitting rating:', error);
        showToast('Failed to submit rating', true);
    }
});

// Add to List
// Add to List
document.getElementById('addToListBtn').addEventListener('click', async () => {
    if (!isLoggedIn()) {
        showToast('Please log in to add books to lists', true);
        return;
    }

    try {
        userLists = await api.getMyLists();

        if (userLists.length === 0) {
            showToast('Create a list first from My Lists page', true);
            return;
        }

        const modal = document.getElementById('addToListModal');
        const listContainer = document.getElementById('listSelectionContainer');

        // Single-select with visual cards
        listContainer.innerHTML = userLists.map(list => `
            <div class="list-selection-card" data-list-id="${list.id}">
                <div class="flex items-start justify-between">
                    <div class="flex-1">
                        <h4 class="font-semibold text-white">${list.name}</h4>
                        ${list.description ? `<p class="text-gray-400 text-sm mt-1">${list.description}</p>` : ''}
                    </div>
                    <span class="text-xs px-2 py-1 rounded ${list.public ? 'bg-blue-500/20 text-blue-400' : 'bg-gray-600 text-gray-300'}">
                        ${list.public ? 'Public' : 'Private'}
                    </span>
                </div>
            </div>
        `).join('');

        // Add click handlers
        document.querySelectorAll('.list-selection-card').forEach(card => {
            card.addEventListener('click', () => {
                document.querySelectorAll('.list-selection-card').forEach(c => c.classList.remove('selected'));
                card.classList.add('selected');
            });
        });

        modal.classList.remove('hidden');
    } catch (error) {
        console.error('Error loading lists:', error);
        showToast('Failed to load lists', true);
    }
});

document.getElementById('closeModal').addEventListener('click', () => {
    document.getElementById('addToListModal').classList.add('hidden');
});

document.getElementById('confirmAddToList').addEventListener('click', async () => {
    const selectedCard = document.querySelector('.list-selection-card.selected');

    if (!selectedCard) {
        showToast('Please select a list', true);
        return;
    }

    const listId = parseInt(selectedCard.dataset.listId);

    try {
        await api.addBookToList(listId, bookId);
        showToast('Book added to list!');
        document.getElementById('addToListModal').classList.add('hidden');
    } catch (error) {
        console.error('Error adding to list:', error);
        showToast('Failed to add book to list', true);
    }
});

// Sort dropdown
document.getElementById('sortSelect').addEventListener('change', (e) => {
    loadRatings(e.target.value);
});

// Like/Unlike
window.toggleLike = async function(ratingId) {
    if (!isLoggedIn()) {
        showToast('Please log in to like reviews', true);
        return;
    }

    try {
        const icon = document.getElementById(`like-icon-${ratingId}`);
        const count = document.getElementById(`like-count-${ratingId}`);
        const isLiked = icon.textContent === '❤️';

        if (isLiked) {
            await api.unlikeRating(ratingId);
            icon.textContent = '🤍';
            count.textContent = parseInt(count.textContent) - 1;
        } else {
            await api.likeRating(ratingId);
            icon.textContent = '❤️';
            count.textContent = parseInt(count.textContent) + 1;
        }
    } catch (error) {
        console.error('Error toggling like:', error);
    }
};

// Comments
window.toggleComments = async function(ratingId) {
    const commentsDiv = document.getElementById(`comments-${ratingId}`);
    const isHidden = commentsDiv.classList.contains('hidden');

    if (isHidden) {
        commentsDiv.classList.remove('hidden');
        await loadComments(ratingId);
    } else {
        commentsDiv.classList.add('hidden');
    }
};

async function loadComments(ratingId) {
    try {
        const comments = await api.getComments(ratingId);
        const container = document.getElementById(`comments-list-${ratingId}`);

        if (comments.length === 0) {
            container.innerHTML = '<p class="text-gray-500 text-sm">No comments yet</p>';
            return;
        }

        container.innerHTML = comments.map(comment => {
            const isOwn = currentUserId && comment.user_id === currentUserId;
            return `
                <div class="mb-3 pb-3 border-b border-gray-700">
                    <div class="flex justify-between items-start">
                        <div>
                            <a href="user-profile.html?id=${comment.user_id}" class="text-blue-400 text-sm font-semibold">${comment.username}</a>
                            <p class="text-gray-300 text-sm mt-1">${comment.text}</p>
                            <span class="text-gray-500 text-xs">${new Date(comment.created_at).toLocaleDateString()}</span>
                        </div>
                        ${isOwn ? `
                            <button onclick="deleteComment(${comment.id}, ${ratingId})" 
                                    class="text-red-400 hover:text-red-300 text-xs">Delete</button>
                        ` : ''}
                    </div>
                </div>
            `;
        }).join('');
    } catch (error) {
        console.error('Error loading comments:', error);
    }
}

window.addComment = async function(ratingId) {
    const input = document.getElementById(`comment-input-${ratingId}`);
    const text = input.value.trim();

    if (!text) {
        showToast('Please enter a comment', true);
        return;
    }

    try {
        await api.createComment(ratingId, text);
        input.value = '';
        await loadComments(ratingId);

        const countEl = document.getElementById(`comment-count-${ratingId}`);
        countEl.textContent = parseInt(countEl.textContent) + 1;
        showToast('Comment added!');
    } catch (error) {
        console.error('Error adding comment:', error);
        showToast('Failed to add comment', true);
    }
};

window.deleteComment = async function(commentId, ratingId) {
    if (!confirm('Delete this comment?')) return;

    try {
        await api.deleteComment(commentId);
        await loadComments(ratingId);

        const countEl = document.getElementById(`comment-count-${ratingId}`);
        countEl.textContent = parseInt(countEl.textContent) - 1;
        showToast('Comment deleted');
    } catch (error) {
        console.error('Error deleting comment:', error);
        showToast('Failed to delete comment', true);
    }
};

// Initialize
loadBookDetails();
loadRatings();
loadSimilarBooks();
loadUserLists();