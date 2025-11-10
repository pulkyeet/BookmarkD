import { api, updateNavigation, getCurrentUserId } from './api.js';

updateNavigation();

const listId = new URLSearchParams(window.location.search).get('id');
const currentUserId = getCurrentUserId();
let books = [];

if (!listId) {
    window.location.href = 'my-lists.html';
}

function showToast(message, isError = false) {
    const toast = document.getElementById('toast');
    toast.textContent = message;
    toast.classList.toggle('error', isError);
    toast.classList.add('show');
    setTimeout(() => toast.classList.remove('show'), 3000);
}

async function loadList() {
    const loading = document.getElementById('loading');
    const listDetail = document.getElementById('listDetail');

    try {
        const list = await api.getList(listId);

        // Check ownership - convert to int for comparison
        if (list.user_id !== parseInt(currentUserId)) {
            loading.classList.add('hidden');
            showToast('You can only view your own lists here', true);
            setTimeout(() => window.location.href = 'my-lists.html', 3000);
            return;
        }

        document.getElementById('listName').textContent = list.name;

        if (list.description) {
            document.getElementById('listDescription').textContent = list.description;
        }

        document.getElementById('listVisibility').textContent = list.public ? '🌐 Public' : '🔒 Private';
        document.getElementById('listCreated').textContent = `Created ${new Date(list.created_at).toLocaleDateString()}`;
        document.getElementById('listCreator').textContent = `by ${list.username}`;

        if (list.public) {
            const embedBtn = document.getElementById('embedBtn');
            embedBtn.classList.remove('hidden');
            embedBtn.addEventListener('click', () => {
                window.location.href = `embed.html?list=${listId}`;
            });
        }

        books = list.books || [];

        loading.classList.add('hidden');
        listDetail.classList.remove('hidden');

        if (books.length === 0) {
            document.getElementById('emptyList').classList.remove('hidden');
            document.getElementById('booksContainer').classList.add('hidden');
        } else {
            document.getElementById('emptyList').classList.add('hidden');
            document.getElementById('booksContainer').classList.remove('hidden');
            document.getElementById('bookCount').textContent = `${books.length} book${books.length !== 1 ? 's' : ''}`;
            renderBooks();
        }

    } catch (error) {
        console.error('Error loading list:', error);
        loading.classList.add('hidden');
        showToast('Failed to load list', true);
        setTimeout(() => window.location.href = 'my-lists.html', 2000);
    }
}

function renderBooks() {
    const container = document.getElementById('booksList');

    container.innerHTML = books.map((book, index) => `
        <div class="book-item" draggable="true" data-book-id="${book.book_id}" data-position="${book.position}">
            <div class="flex gap-4">
                <div class="flex-shrink-0">
                    <span class="text-gray-500 font-bold text-lg">${index + 1}</span>
                </div>
                <img src="${book.cover_url || 'https://via.placeholder.com/80x120'}" 
                     alt="${book.title}" 
                     class="w-20 h-28 object-cover rounded">
                <div class="flex-1">
                    <h3 class="font-bold text-lg mb-1">${book.title}</h3>
                    <p class="text-gray-400 text-sm mb-2">${book.author}</p>
                    <p class="text-gray-500 text-xs">Added ${new Date(book.added_at).toLocaleDateString()}</p>
                </div>
                <div class="flex flex-col gap-2">
                    <button onclick="viewBook(${book.book_id})" class="btn-secondary text-sm px-4 py-2">View</button>
                    <button onclick="removeBook(${book.book_id})" class="text-red-400 hover:text-red-300 text-sm px-4 py-2">🗑️</button>
                </div>
            </div>
        </div>
    `).join('');

    setupDragAndDrop();
}

function setupDragAndDrop() {
    const items = document.querySelectorAll('.book-item');

    items.forEach(item => {
        item.addEventListener('dragstart', handleDragStart);
        item.addEventListener('dragover', handleDragOver);
        item.addEventListener('drop', handleDrop);
        item.addEventListener('dragenter', handleDragEnter);
        item.addEventListener('dragleave', handleDragLeave);
        item.addEventListener('dragend', handleDragEnd);
    });
}

let draggedElement = null;

function handleDragStart(e) {
    draggedElement = this;
    this.classList.add('dragging');
    e.dataTransfer.effectAllowed = 'move';
}

function handleDragOver(e) {
    if (e.preventDefault) {
        e.preventDefault();
    }
    e.dataTransfer.dropEffect = 'move';
    return false;
}

function handleDragEnter(e) {
    if (this !== draggedElement) {
        this.classList.add('drag-over');
    }
}

function handleDragLeave(e) {
    this.classList.remove('drag-over');
}

function handleDrop(e) {
    if (e.stopPropagation) {
        e.stopPropagation();
    }

    this.classList.remove('drag-over');

    if (draggedElement !== this) {
        // Reorder in array
        const draggedBookId = parseInt(draggedElement.dataset.bookId);
        const targetBookId = parseInt(this.dataset.bookId);

        const draggedIndex = books.findIndex(b => b.book_id === draggedBookId);
        const targetIndex = books.findIndex(b => b.book_id === targetBookId);

        const [removed] = books.splice(draggedIndex, 1);
        books.splice(targetIndex, 0, removed);

        // Update positions
        books.forEach((book, index) => {
            book.position = index + 1;
        });

        renderBooks();
        saveOrder();
    }

    return false;
}

function handleDragEnd(e) {
    this.classList.remove('dragging');
    document.querySelectorAll('.book-item').forEach(item => {
        item.classList.remove('drag-over');
    });
}

async function saveOrder() {
    try {
        const bookPositions = books.map(book => ({
            book_id: book.book_id,
            position: book.position
        }));

        await api.reorderListBooks(listId, bookPositions);
        showToast('Order saved!');
    } catch (error) {
        console.error('Error saving order:', error);
        showToast('Failed to save order', true);
    }
}

window.viewBook = function(bookId) {
    window.location.href = `book-detail.html?id=${bookId}`;
};

window.removeBook = async function(bookId) {
    if (!confirm('Remove this book from the list?')) return;

    try {
        await api.removeBookFromList(listId, bookId);
        books = books.filter(b => b.book_id !== bookId);
        showToast('Book removed');

        if (books.length === 0) {
            document.getElementById('emptyList').classList.remove('hidden');
            document.getElementById('booksContainer').classList.add('hidden');
        } else {
            document.getElementById('bookCount').textContent = `${books.length} book${books.length !== 1 ? 's' : ''}`;
            renderBooks();
        }
    } catch (error) {
        console.error('Error removing book:', error);
        showToast('Failed to remove book', true);
    }
};

// Initialize
loadList();