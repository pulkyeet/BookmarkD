import { api, updateNavigation } from './api.js';

updateNavigation();

let currentPage = 0;
let currentSearch = '';
const limit = 20;

async function loadBooks() {
    const loading = document.getElementById('loading');
    const grid = document.getElementById('booksGrid');
    const pagination = document.getElementById('pagination');

    loading.classList.remove('hidden');
    grid.classList.add('hidden');

    try {
        const offset = currentPage * limit;
        const books = await api.getBooks({ limit, offset, search: currentSearch });

        loading.classList.add('hidden');
        grid.classList.remove('hidden');
        pagination.classList.remove('hidden');

        grid.innerHTML = books.map(book => `
            <div class="book-card" onclick="window.location.href='book-detail.html?id=${book.id}'">
                <img src="${book.cover_url || 'https://via.placeholder.com/300x450?text=No+Cover'}" 
                     alt="${book.title}" class="book-cover">
                <div class="p-4">
                    <h3 class="font-bold text-lg mb-1 line-clamp-2">${book.title}</h3>
                    <p class="text-gray-400 text-sm mb-2">${book.author}</p>
                    ${book.published_year ? `<p class="text-gray-500 text-xs">${book.published_year}</p>` : ''}
                </div>
            </div>
        `).join('');

        document.getElementById('pageInfo').textContent = `Page ${currentPage + 1}`;
        document.getElementById('prevBtn').disabled = currentPage === 0;
        document.getElementById('nextBtn').disabled = books.length < limit;

    } catch (error) {
        console.error('Error loading books:', error);
        loading.innerHTML = '<p class="text-red-400">Failed to load books</p>';
    }
}

document.getElementById('searchBtn').addEventListener('click', () => {
    currentSearch = document.getElementById('searchInput').value;
    currentPage = 0;
    loadBooks();
});

document.getElementById('searchInput').addEventListener('keypress', (e) => {
    if (e.key === 'Enter') {
        currentSearch = e.target.value;
        currentPage = 0;
        loadBooks();
    }
});

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

loadBooks();