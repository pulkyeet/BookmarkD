import { api, isLoggedIn, updateNavigation, logout } from './api.js';

if (!isLoggedIn()) {
    window.location.href = 'login.html';
}

updateNavigation();

let currentStatus = 'all';

async function loadProfile(status = '') {
    const loading = document.getElementById('loading');
    const ratingsList = document.getElementById('ratingsList');
    const noRatings = document.getElementById('noRatings');

    try {
        const [profile, ratings] = await Promise.all([
            api.getProfile(),
            api.getMyRatings()
        ]);

        document.getElementById('username').textContent = profile.username;
        document.getElementById('email').textContent = profile.email;

        loading.classList.add('hidden');

        // Filter ratings by status
        let filteredRatings = ratings;
        if (status && status !== 'all') {
            filteredRatings = ratings.filter(r => r.status === status);
        }

        if (filteredRatings && filteredRatings.length > 0) {
            document.getElementById('totalRatings').textContent = ratings.length;
            const avgRating = ratings.length > 0
                ? (ratings.reduce((sum, r) => sum + r.rating, 0) / ratings.length).toFixed(1)
                : '0.0';
            document.getElementById('avgRating').textContent = avgRating;

            ratingsList.classList.remove('hidden');
            noRatings.classList.add('hidden');

            // Load book details for each rating
            const ratingsWithBooks = await Promise.all(
                filteredRatings.map(async (rating) => {
                    try {
                        const book = await api.getBook(rating.book_id);
                        return { ...rating, book };
                    } catch (err) {
                        console.error(`Failed to load book ${rating.book_id}:`, err);
                        return null;
                    }
                })
            );

            const validRatingsWithBooks = ratingsWithBooks.filter(r => r !== null);
            ratingsList.innerHTML = validRatingsWithBooks.map(r => `
                <div class="review-card cursor-pointer hover:border-blue-500 transition"
                     onclick="window.location.href='book-detail.html?id=${r.book_id}'">
                    <div class="flex gap-4">
                        <img src="${r.book.cover_url || 'https://via.placeholder.com/100x150'}" 
                             alt="${r.book.title}" 
                             class="w-20 h-30 object-cover rounded">
                        <div class="flex-1">
                            <h3 class="font-bold text-lg mb-1">${r.book.title}</h3>
                            <p class="text-gray-400 text-sm mb-2">${r.book.author}</p>
                            <div class="flex gap-2 items-center mb-2">
                                ${r.rating > 0 ? `<div class="rating-badge">${r.rating}/10</div>` : ''}
                                <span class="text-xs text-gray-500">${getStatusLabel(r.status)}</span>
                            </div>
                            ${r.review ? `<p class="text-gray-400 text-sm">${r.review}</p>` : ''}
                            <p class="text-xs text-gray-500 mt-2">${new Date(r.created_at).toLocaleDateString()}</p>
                        </div>
                    </div>
                </div>
            `).join('');
        } else {
            ratingsList.classList.add('hidden');
            noRatings.classList.remove('hidden');
        }

    } catch (error) {
        console.error('Error loading profile:', error);
        loading.classList.add('hidden');
        loading.innerHTML = `<p class="text-red-400">Failed to load profile: ${error.message}</p>`;

        if (error.message.includes('Unauthorized') || error.message.includes('401')) {
            localStorage.removeItem('token');
            window.location.href = 'login.html';
        }
    }
}

function getStatusLabel(status) {
    switch(status) {
        case 'to_read': return '📚 To Read';
        case 'currently_reading': return '📖 Reading';
        case 'finished_reading': return '✅ Finished';
        default: return '';
    }
}

// Tab switching
document.querySelectorAll('.tab-btn').forEach(btn => {
    btn.addEventListener('click', () => {
        currentStatus = btn.dataset.status;

        // Update active tab
        document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');

        // Update section title
        const titles = {
            'all': 'Your Books',
            'to_read': 'Want to Read',
            'currently_reading': 'Currently Reading',
            'finished_reading': 'Finished Books'
        };
        document.getElementById('sectionTitle').textContent = titles[currentStatus];

        // Reload with filter
        loadProfile(currentStatus === 'all' ? '' : currentStatus);
    });
});

loadProfile();