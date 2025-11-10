const API_BASE = 'http://localhost:8080/api';
const SITE_BASE = 'http://localhost:8080';

function getAuthToken() {
    return localStorage.getItem('token');
}

const state = {
    userId: null,
    sourceType: 'top_rated',
    listId: null,
    count: 5,
    style: 'grid',
    lists: []
};

async function init() {
    const token = getAuthToken();
    if (!token) {
        window.location.href = '/login.html';
        return;
    }

    try {
        const payload = JSON.parse(atob(token.split('.')[1]));
        state.userId = payload.user_id;
    } catch (err) {
        window.location.href = '/login.html';
        return;
    }

    // Check for pre-selected list in URL
    const urlParams = new URLSearchParams(window.location.search);
    const preselectedListId = urlParams.get('list');
    if (preselectedListId) {
        state.sourceType = 'list';
        state.listId = parseInt(preselectedListId);
        document.getElementById('sourceType').value = 'list';
        document.getElementById('listSelector').classList.remove('hidden');
    }

    await loadUserLists();
    setupEventListeners();
}

async function loadUserLists() {
    try {
        const res = await fetch(`${API_BASE}/users/${state.userId}/lists`, {
            headers: { 'Authorization': `Bearer ${getAuthToken()}` }
        });
        if (res.ok) {
            state.lists = await res.json();
            renderListOptions();
        }
    } catch (err) {
        console.error('Failed to load lists:', err);
    }
}

function renderListOptions() {
    const select = document.getElementById('listSelect');
    const publicLists = state.lists.filter(list => list.public);

    if (publicLists.length === 0) {
        select.innerHTML = '<option value="">No public lists found</option>';
    } else {
        select.innerHTML = publicLists
            .map(list => `<option value="${list.id}">${list.name}</option>`)
            .join('');

        // Use pre-selected list if available, otherwise first list
        if (!state.listId || !publicLists.find(l => l.id === state.listId)) {
            state.listId = publicLists[0].id;
        }
        select.value = state.listId;
    }
}

function setupEventListeners() {
    // Logout
    document.getElementById('logoutBtn').addEventListener('click', () => {
        localStorage.removeItem('token');
        window.location.href = '/login.html';
    });

    // Source type change
    document.getElementById('sourceType').addEventListener('change', (e) => {
        state.sourceType = e.target.value;
        const listSelector = document.getElementById('listSelector');
        if (state.sourceType === 'list') {
            listSelector.classList.remove('hidden');
        } else {
            listSelector.classList.add('hidden');
        }
        updatePreview();
    });

    // List selection
    document.getElementById('listSelect').addEventListener('change', (e) => {
        state.listId = parseInt(e.target.value);
        updatePreview();
    });

    // Count change
    document.getElementById('bookCount').addEventListener('change', (e) => {
        state.count = parseInt(e.target.value);
        updatePreview();
    });

    // Style buttons
    document.querySelectorAll('.style-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.style-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            state.style = btn.dataset.style;
            updatePreview();
        });
    });

    // Generate button
    document.getElementById('generateBtn').addEventListener('click', () => {
        generateCode();
    });

    // Code tabs
    document.querySelectorAll('.code-tab').forEach(tab => {
        tab.addEventListener('click', () => {
            document.querySelectorAll('.code-tab').forEach(t => t.classList.remove('active'));
            tab.classList.add('active');

            if (tab.dataset.tab === 'iframe') {
                document.getElementById('iframeCode').classList.remove('hidden');
                document.getElementById('scriptCode').classList.add('hidden');
            } else {
                document.getElementById('iframeCode').classList.add('hidden');
                document.getElementById('scriptCode').classList.remove('hidden');
            }
        });
    });

    // Copy buttons
    document.getElementById('copyIframe').addEventListener('click', () => {
        copyToClipboard(document.getElementById('iframeCodeContent').textContent);
    });
    document.getElementById('copyScript').addEventListener('click', () => {
        copyToClipboard(document.getElementById('scriptCodeContent').textContent);
    });

    // Initial preview after DOM is ready
    setTimeout(() => updatePreview(), 200);
}

function updatePreview() {
    const iframe = document.getElementById('previewFrame');
    let url;

    if (state.sourceType === 'top_rated') {
        url = `${SITE_BASE}/embed-user.html?user=${state.userId}&count=${state.count}&style=${state.style}`;
    } else {
        if (!state.listId) return;
        url = `${SITE_BASE}/embed-list.html?list=${state.listId}&count=${state.count}&style=${state.style}`;
    }

    iframe.src = url;
}

function generateCode() {
    let embedUrl;
    let dataType, dataId;

    if (state.sourceType === 'top_rated') {
        embedUrl = `${SITE_BASE}/embed-user.html?user=${state.userId}&count=${state.count}&style=${state.style}`;
        dataType = 'user';
        dataId = state.userId;
    } else {
        if (!state.listId) {
            showToast('Please select a list', 'error');
            return;
        }
        embedUrl = `${SITE_BASE}/embed-list.html?list=${state.listId}&count=${state.count}&style=${state.style}`;
        dataType = 'list';
        dataId = state.listId;
    }

    // Iframe code
    const iframeCode = `<iframe src="${embedUrl}" width="100%" height="500" frameborder="0"></iframe>`;
    document.getElementById('iframeCodeContent').textContent = iframeCode;

    // Script tag code - now uses bookmarkd-embed.js
    const scriptCode = `<!-- BookmarkD Embed -->
<div 
    data-bookmarkd-embed="${dataType}" 
    data-bookmarkd-id="${dataId}"
    data-bookmarkd-count="${state.count}"
    data-bookmarkd-style="${state.style}">
</div>
<script src="${SITE_BASE}/bookmarkd-embed.js"></script>`;
    document.getElementById('scriptCodeContent').textContent = scriptCode;

    document.getElementById('codeOutput').classList.remove('hidden');
    showToast('Embed code generated!');
}

function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
        showToast('Copied to clipboard!');
    }).catch(err => {
        showToast('Failed to copy', 'error');
    });
}

function showToast(message, type = 'success') {
    const toast = document.getElementById('toast');
    toast.textContent = message;
    toast.className = `toast show ${type === 'error' ? 'error' : ''}`;
    setTimeout(() => toast.classList.remove('show'), 3000);
}

init();