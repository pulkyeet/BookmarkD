import { api, isLoggedIn, updateNavigation } from './api.js';

updateNavigation();

const userId = new URLSearchParams(window.location.search).get('id');
if (!userId) {
    window.location.href = 'feed.html';
}

function showToast(message) {
    const toast = document.getElementById('toast');
    toast.textContent = message;
    toast.classList.add('show');
    setTimeout(() => toast.classList.remove('show'), 3000);
}

async function loadProfile() {
    const loading = document.getElementById('loading');
    const content = document.getElementById('profileContent');

    try {
        const profile = await api.getUserProfile(userId);

        loading.classList.add('hidden');
        content.classList.remove('hidden');

        document.getElementById('username').textContent = profile.username;
        document.getElementById('email').textContent = profile.email;
        document.getElementById('totalBooks').textContent = profile.total_books;
        document.getElementById('avgRating').textContent = profile.average_rating.toFixed(1);
        document.getElementById('toReadCount').textContent = profile.to_read_count;
        document.getElementById('currentlyReading').textContent = profile.currently_reading_count;
        document.getElementById('finishedCount').textContent = profile.finished_count;
        document.getElementById('followersCount').textContent = profile.followers_count;
        document.getElementById('followingCount').textContent = profile.following_count;

        // Make follower/following counts clickable
        document.getElementById('followersBtn').addEventListener('click', () => showFollowModal('followers'));
        document.getElementById('followingBtn').addEventListener('click', () => showFollowModal('following'));

        // Show follow button only if logged in and not own profile
        if (isLoggedIn()) {
            const myProfile = await api.getProfile();
            if (myProfile.user_id !== parseInt(userId)) {
                const followBtn = document.getElementById('followBtn');
                followBtn.classList.remove('hidden');
                updateFollowButton(profile.is_following);

                followBtn.addEventListener('click', async () => {
                    try {
                        if (profile.is_following) {
                            await api.unfollowUser(userId);
                            profile.is_following = false;
                            showToast('Unfollowed!');
                        } else {
                            await api.followUser(userId);
                            profile.is_following = true;
                            showToast('Followed!');
                        }
                        updateFollowButton(profile.is_following);
                        setTimeout(() => location.reload(), 1000);
                    } catch (error) {
                        showToast('Error: ' + error.message);
                    }
                });
            }
        }

    } catch (error) {
        console.error('Error loading profile:', error);
        loading.classList.add('hidden');
        content.innerHTML = `<p class="text-red-400">Failed to load profile</p>`;
        content.classList.remove('hidden');
    }
}

async function showFollowModal(type) {
    const modal = document.getElementById('followModal');
    const modalTitle = document.getElementById('modalTitle');
    const modalContent = document.getElementById('modalContent');

    modalTitle.textContent = type === 'followers' ? 'Followers' : 'Following';
    modalContent.innerHTML = '<div class="text-center py-4"><div class="spinner"></div></div>';
    modal.classList.remove('hidden');
    modal.classList.add('flex');

    try {
        const users = type === 'followers'
            ? await api.getFollowers(userId)
            : await api.getFollowing(userId);

        if (users && users.length > 0) {
            modalContent.innerHTML = users.map(user => `
                <div class="flex items-center justify-between p-3 bg-gray-700/50 rounded-lg hover:bg-gray-700 transition cursor-pointer"
                     onclick="window.location.href='user-profile.html?id=${user.id}'">
                    <div>
                        <p class="font-bold">${user.username}</p>
                        <p class="text-sm text-gray-400">${user.email}</p>
                    </div>
                    <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path>
                    </svg>
                </div>
            `).join('');
        } else {
            modalContent.innerHTML = `<p class="text-gray-400 text-center py-8">No ${type} yet</p>`;
        }
    } catch (error) {
        console.error('Error loading ' + type, error);
        modalContent.innerHTML = `<p class="text-red-400 text-center">Failed to load ${type}</p>`;
    }
}

window.closeModal = function() {
    const modal = document.getElementById('followModal');
    modal.classList.add('hidden');
    modal.classList.remove('flex');
}

function updateFollowButton(isFollowing) {
    const followBtn = document.getElementById('followBtn');
    if (isFollowing) {
        followBtn.textContent = 'Unfollow';
        followBtn.classList.remove('btn-primary');
        followBtn.classList.add('btn-secondary');
    } else {
        followBtn.textContent = 'Follow';
        followBtn.classList.remove('btn-secondary');
        followBtn.classList.add('btn-primary');
    }
}

loadProfile();