import { api, updateNavigation, getCurrentUserId } from './api.js';

updateNavigation();

let editingListId = null;

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
        const lists = await api.getMyLists();

        loading.classList.add('hidden');

        if (lists.length === 0) {
            emptyState.classList.remove('hidden');
            return;
        }

        grid.classList.remove('hidden');
        grid.innerHTML = lists.map(list => `
            <div class="list-card">
                <div onclick="window.location.href='list-detail.html?id=${list.id}'">
                    <div class="flex justify-between items-start mb-3">
                        <h3 class="text-xl font-bold">${list.name}</h3>
                        <span class="text-xs px-2 py-1 rounded ${list.public ? 'bg-blue-500/20 text-blue-400' : 'bg-gray-600 text-gray-300'}">
                            ${list.public ? 'Public' : 'Private'}
                        </span>
                    </div>
                    ${list.description ? `<p class="text-gray-400 text-sm mb-3">${list.description}</p>` : ''}
                    <p class="text-gray-500 text-xs">Created ${new Date(list.created_at).toLocaleDateString()}</p>
                </div>
                <div class="flex gap-2 mt-4">
                    <button onclick="editList(${list.id}, '${list.name.replace(/'/g, "\\'")}', '${(list.description || '').replace(/'/g, "\\'")}', ${list.public}); event.stopPropagation();" 
                            class="btn-secondary text-sm flex-1">Edit</button>
                    <button onclick="deleteList(${list.id}); event.stopPropagation();" 
                            class="text-red-400 hover:text-red-300 text-sm px-4">🗑️ Delete</button>
                </div>
            </div>
        `).join('');

    } catch (error) {
        console.error('Error loading lists:', error);
        loading.classList.add('hidden');
        showToast('Failed to load lists', true);
    }
}

// Create list button
document.getElementById('createListBtn').addEventListener('click', () => {
    editingListId = null;
    document.getElementById('modalTitle').textContent = 'Create New List';
    document.getElementById('listForm').reset();
    document.getElementById('listPublic').checked = true;
    document.getElementById('listForm').querySelector('button[type="submit"]').textContent = 'Create List';
    document.getElementById('listModal').classList.remove('hidden');
});

// Close modal
document.getElementById('closeModal').addEventListener('click', () => {
    document.getElementById('listModal').classList.add('hidden');
});

// Form submission
document.getElementById('listForm').addEventListener('submit', async (e) => {
    e.preventDefault();

    const name = document.getElementById('listName').value;
    const description = document.getElementById('listDescription').value;
    const isPublic = document.getElementById('listPublic').checked;

    try {
        if (editingListId) {
            await api.updateList(editingListId, name, description, isPublic);
            showToast('List updated!');
        } else {
            await api.createList(name, description, isPublic);
            showToast('List created!');
        }
        document.getElementById('listModal').classList.add('hidden');
        loadLists();
    } catch (error) {
        console.error('Error saving list:', error);
        showToast('Failed to save list', true);
    }
});

// Edit list
window.editList = function(id, name, description, isPublic) {
    editingListId = id;
    document.getElementById('modalTitle').textContent = 'Edit List';
    document.getElementById('listName').value = name;
    document.getElementById('listDescription').value = description;
    document.getElementById('listPublic').checked = isPublic;
    document.getElementById('listForm').querySelector('button[type="submit"]').textContent = 'Update List';
    document.getElementById('listModal').classList.remove('hidden');
};

// Delete list
window.deleteList = async function(id) {
    if (!confirm('Delete this list? This cannot be undone.')) return;

    try {
        await api.deleteList(id);
        showToast('List deleted');
        loadLists();
    } catch (error) {
        console.error('Error deleting list:', error);
        showToast('Failed to delete list', true);
    }
};

// Initialize
loadLists();