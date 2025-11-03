import { api, updateNavigation, isLoggedIn } from './api.js';

updateNavigation();

// Show My Lists link if logged in
if (isLoggedIn()) {
    document.getElementById('myListsLink')?.classList.remove('hidden');
}

let currentPage = 0;
let currentSearch = '';
let currentGenre = '';
let currentTab = 'all'; // 'all', 'trending', 'popular'
const limit = 20;

// Load genres into dropdown
async function loadGenres() {
    try {
        const genres = await api.getGenres();
        const genreFilter = document.getElementById('genreFilter');

        genres.forEach(genre => {
            const option = document.createElement('option');
            option.value = genre.name;
            option.textContent = genre.name;
            genreFilter.appendChild(option);
        });
    } catch (error) {
        console.error('Error loading genres:', error);
    }
}

async function loadBooks() {
    const loading = document.getElementById('loading');
    const grid = document.getElementById('booksGrid');
    const pagination = document.getElementById('pagination');

    loading.classList.remove('hidden');
    grid.classList.add('hidden');

    try {
        let books;

        if (currentTab === 'trending') {
            books = await api.getTrendingBooks('week', limit);
            pagination.classList.add('hidden'); // No pagination for trending
        } else if (currentTab === 'popular') {
            books = await api.getPopularBooks(5, limit);
            pagination.classList.add('hidden'); // No pagination for popular
        } else {
            const offset = currentPage * limit;
            books = await api.getBooks({
                limit,
                offset,
                search: currentSearch,
                genre: currentGenre
            });
            pagination.classList.remove('hidden');
        }

        loading.classList.add('hidden');
        grid.classList.remove('hidden');

        if (books.length === 0) {
            grid.innerHTML = '<p class="text-gray-400 text-center col-span-full">No books found</p>';
            return;
        }

        grid.innerHTML = books.map(book => {
            const genreBadges = book.genres && book.genres.length > 0
                ? book.genres.slice(0, 3).map(g => `<span class="genre-badge">${g.name}</span>`).join('')
                : '';

            return `
                <div class="book-card cursor-pointer" onclick="window.location.href='book-detail.html?id=${book.id || book.book_id}'">
                    <img src="${book.cover_url || 'https://via.placeholder.com/300x450?text=No+Cover'}" 
                         alt="${book.title}" class="book-cover">
                    <div class="p-4">
                        <h3 class="font-bold text-lg mb-1 line-clamp-2">${book.title}</h3>
                        <p class="text-gray-400 text-sm mb-2">${book.author}</p>
                        ${genreBadges ? `<div class="mb-2">${genreBadges}</div>` : ''}
                        ${book.published_year ? `<p class="text-gray-500 text-xs">${book.published_year}</p>` : ''}
                        ${book.avg_rating ? `<p class="text-blue-400 text-sm mt-2">⭐ ${book.avg_rating.toFixed(1)}</p>` : ''}
                        ${book.rating_count ? `<p class="text-gray-500 text-xs">${book.rating_count} ratings</p>` : ''}
                    </div>
                </div>
            `;
        }).join('');

        if (currentTab === 'all') {
            document.getElementById('pageInfo').textContent = `Page ${currentPage + 1}`;
            document.getElementById('prevBtn').disabled = currentPage === 0;
            document.getElementById('nextBtn').disabled = books.length < limit;
        }

    } catch (error) {
        console.error('Error loading books:', error);
        loading.classList.add('hidden');
        grid.innerHTML = '<p class="text-red-400 text-center col-span-full">Failed to load books</p>';
    }
}

// Tab switching
document.getElementById('allTab').addEventListener('click', () => {
    currentTab = 'all';
    currentPage = 0;
    document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
    document.getElementById('allTab').classList.add('active');
    loadBooks();
});

document.getElementById('trendingTab').addEventListener('click', () => {
    currentTab = 'trending';
    document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
    document.getElementById('trendingTab').classList.add('active');
    loadBooks();
});

document.getElementById('popularTab').addEventListener('click', () => {
    currentTab = 'popular';
    document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
    document.getElementById('popularTab').classList.add('active');
    loadBooks();
});

// Search and filter
document.getElementById('searchBtn').addEventListener('click', () => {
    currentSearch = document.getElementById('searchInput').value;
    currentGenre = document.getElementById('genreFilter').value;
    currentTab = 'all';
    currentPage = 0;
    document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
    document.getElementById('allTab').classList.add('active');
    loadBooks();
});

document.getElementById('searchInput').addEventListener('keypress', (e) => {
    if (e.key === 'Enter') {
        currentSearch = e.target.value;
        currentGenre = document.getElementById('genreFilter').value;
        currentTab = 'all';
        currentPage = 0;
        document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
        document.getElementById('allTab').classList.add('active');
        loadBooks();
    }
});

document.getElementById('genreFilter').addEventListener('change', (e) => {
    currentGenre = e.target.value;
    currentTab = 'all';
    currentPage = 0;
    document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
    document.getElementById('allTab').classList.add('active');
    loadBooks();
});

// Pagination
document.getElementById('prevBtn').addEventListener('click', () => {
    if (currentPage > 0) {
        currentPage--;
        loadBooks();
    }
});

document.getElementById('nextBtn').addEventListener('click', () => {
    currentPage++;
    loadBooks();
});

// Initialize
loadGenres();
loadBooks();