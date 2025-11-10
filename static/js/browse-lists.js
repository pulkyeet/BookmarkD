import { api, updateNavigation, isLoggedIn } from './api.js';

updateNavigation();

let currentTab = 'popular';

if (isLoggedIn()) {
    document.getElementById('myListsLink')?.classList.remove('hidden');
    document.getElementById('bookmarkedTab')?.classList.remove('hidden');
}

function showToast(message, isError = false) {
    const toast = document.getElementById('toast');
    toast.textContent = message;
    toast.classList.toggle('error', isError);
    toast.classList.add('show');
    setTimeout(() => toast.classList.remove('show'), 3000);
}

async function loadLists() {
    const loading = document.getElementById('loading');
    const grid = document.getElementById('listsGrid');
    const emptyState = document.getElementById('emptyState');

    loading.classList.remove('hidden');
    grid.classList.add('hidden');
    emptyState.classList.add('hidden');

    try {
        let lists;

        if (currentTab === 'popular') {
            lists = await api.getPopularLists(20);
        } else {
            lists = await api.getBookmarkedLists();
        }

        loading.classList.add('hidden');

        if (lists.length === 0) {
            emptyState.classList.remove('hidden');
            return;
        }

        grid.classList.remove('hidden');
        grid.innerHTML = lists.map(list => `
            <div class="list-card">
                <div class="cursor-pointer" onclick="viewList(${list.id})">
                    <div class="flex justify-between items-start mb-3">
                        <h3 class="text-xl font-bold">${list.name}</h3>
                    </div>
                    ${list.description ? `<p class="text-gray-400 text-sm mb-3">${list.description}</p>` : ''}
                    <p class="text-gray-500 text-xs mb-3">Created ${new Date(list.created_at).toLocaleDateString()}</p>
                </div>
                ${isLoggedIn() ? `
                    <button onclick="toggleBookmark(${list.id}); event.stopPropagation();" 
                            id="bookmark-btn-${list.id}"
                            class="btn-secondary w-full text-sm">
                        ${currentTab === 'bookmarked' ? 'Remove Bookmark' : '+ Bookmark'}
                    </button>
                ` : ''}
            </div>
        `).join('');

    } catch (error) {
        console.error('Error loading lists:', error);
        loading.classList.add('hidden');
        showToast('Failed to load lists', true);
    }
}

// Tab switching
document.getElementById('popularTab').addEventListener('click', () => {
    currentTab = 'popular';
    document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
    document.getElementById('popularTab').classList.add('active');
    loadLists();
});

document.getElementById('bookmarkedTab')?.addEventListener('click', () => {
    currentTab = 'bookmarked';
    document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
    document.getElementById('bookmarkedTab').classList.add('active');
    loadLists();
});

window.viewList = function(listId) {
    window.location.href = `list-detail.html?id=${listId}`;
};

window.toggleBookmark = async function(listId) {
    if (!isLoggedIn()) {
        showToast('Please log in to bookmark lists', true);
        return;
    }

    try {
        if (currentTab === 'bookmarked') {
            await api.unbookmarkList(listId);
            showToast('Bookmark removed');
        } else {
            await api.bookmarkList(listId);
            showToast('List bookmarked!');
        }
        loadLists();
    } catch (error) {
        console.error('Error toggling bookmark:', error);
        showToast('Failed to update bookmark', true);
    }
};

// Initialize
loadLists();