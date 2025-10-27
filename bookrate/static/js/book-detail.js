import { api, isLoggedIn, updateNavigation } from './api.js';

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

async function loadBook() {
    const loading = document.getElementById('loading');
    const detail = document.getElementById('bookDetail');

    try {
        const [book, ratings] = await Promise.all([
            api.getBook(bookId),
            api.getRatings(bookId)
        ]);

        loading.classList.add('hidden');
        detail.classList.remove('hidden');

        // Book details
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

        // Ratings
        const avgRating = ratings.average_rating || 0;
        document.getElementById('avgRating').textContent = avgRating.toFixed(1) + '/10';
        document.getElementById('totalRatings').textContent = `${ratings.total_ratings} rating${ratings.total_ratings !== 1 ? 's' : ''}`;

        // Reviews - HIDE 0 ratings
        const reviewsList = document.getElementById('reviewsList');
        const noReviews = document.getElementById('noReviews');

        if (ratings.ratings && ratings.ratings.length > 0) {
            reviewsList.innerHTML = ratings.ratings.map(r => {
                const ratingValue = r.rating?.rating || r.rating;
                return `
                <div class="review-card">
                    <div class="flex justify-between items-start mb-2">
                        <div>
                            <p class="font-bold cursor-pointer hover:text-blue-400" 
                               onclick="window.location.href='user-profile.html?id=${r.user_id}'">
                                ${r.username}
                            </p>
                            ${ratingValue > 0 ? `<div class="rating-badge">${ratingValue}/10</div>` : ''}
                        </div>
                        <p class="text-sm text-gray-500">${new Date(r.created_at).toLocaleDateString()}</p>
                    </div>
                    ${r.review ? `<p class="text-gray-400 mt-2">${r.review}</p>` : ''}
                </div>
            `}).join('');
        } else {
            noReviews.classList.remove('hidden');
        }

        // Show rating form or login prompt
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

async function loadExistingRating() {
    try {
        existingRating = await api.getMyRatingForBook(bookId);

        if (existingRating) {
            // Pre-select status
            selectedStatus = existingRating.status;
            const statusBtn = document.querySelector(`.status-btn[data-status="${existingRating.status}"]`);
            if (statusBtn) {
                statusBtn.classList.add('selected');
            }

            // Pre-select rating if > 0
            if (existingRating.rating > 0) {
                selectedRating = existingRating.rating;
                document.getElementById('ratingValue').value = selectedRating;
                const ratingBtn = document.querySelector(`.rating-btn[data-rating="${existingRating.rating}"]`);
                if (ratingBtn) {
                    ratingBtn.classList.add('selected');
                }
            }

            // Pre-fill review
            if (existingRating.review) {
                document.getElementById('reviewText').value = existingRating.review;
            }
        }
    } catch (error) {
        // 404 means no rating exists, which is fine
        if (!error.message.includes('404')) {
            console.error('Error loading existing rating:', error);
        }
    }
}

function setupRatingForm() {
    // Status buttons
    const statusBtns = document.querySelectorAll('.status-btn');
    statusBtns.forEach(btn => {
        btn.addEventListener('click', async () => {
            selectedStatus = btn.dataset.status;
            statusBtns.forEach(b => b.classList.remove('selected'));
            btn.classList.add('selected');

            // For to_read and currently_reading, save immediately without rating
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

    // Rating buttons
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
            showToast('Rating submitted! 🎉', 'success');
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