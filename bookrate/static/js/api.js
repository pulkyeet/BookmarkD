const API_BASE = 'http://localhost:8080/api';

export const api = {
    async request(endpoint, options = {}) {
        const token = localStorage.getItem('token');

        const headers = {
            'Content-Type': 'application/json',
            ...options.headers,
        };

        if (token) {
            headers['Authorization'] = `Bearer ${token}`;
        }

        try {
            const response = await fetch(`${API_BASE}${endpoint}`, {
                ...options,
                headers,
            });

            // Handle empty responses (like 204 No Content for delete)
            if (response.status === 204) {
                return null;
            }

            const contentType = response.headers.get('content-type');

            if (!response.ok) {
                let errorMessage = `Request failed with status ${response.status}`;

                if (contentType && contentType.includes('application/json')) {
                    const errorData = await response.json();
                    errorMessage = errorData.message || errorData.error || errorMessage;
                } else {
                    errorMessage = await response.text();
                }

                throw new Error(errorMessage);
            }

            if (contentType && contentType.includes('application/json')) {
                return response.json();
            }

            return response.text();

        } catch (error) {
            console.error('API Error:', error);
            throw error;
        }
    },

    // Auth
    async login(email, password) {
        return this.request('/auth/login', {
            method: 'POST',
            body: JSON.stringify({ email, password }),
        });
    },

    async signup(username, email, password) {
        return this.request('/auth/signup', {
            method: 'POST',
            body: JSON.stringify({ username, email, password }),
        });
    },

    // Books
    async getBooks(params = {}) {
        const query = new URLSearchParams(params).toString();
        return this.request(`/books?${query}`);
    },

    async getBook(id) {
        return this.request(`/books/${id}`);
    },

    // Ratings
    async getRatings(bookId) {
        return this.request(`/books/${bookId}/ratings`);
    },

    async createRating(bookId, rating, review, status = 'finished_reading') {
        return this.request(`/books/${bookId}/ratings`, {
            method: 'POST',
            body: JSON.stringify({ rating, review, status }),
        });
    },

    async deleteRating(bookId) {
        return this.request(`/books/${bookId}/ratings`, {
            method: 'DELETE',
        });
    },

    async getMyRatings() {
        return this.request('/users/me/ratings');
    },

    async getProfile() {
        return this.request('/profile');
    },

    async getMyRatingForBook(bookId) {
        return this.request(`/books/${bookId}/ratings/me`);
    },

    async getUserProfile(userId) {
        return this.request(`/users/${userId}/profile`);
    },

    async followUser(userId) {
        return this.request(`/users/${userId}/follow`, {
            method: 'POST',
        });
    },

    async unfollowUser(userId) {
        return this.request(`/users/${userId}/follow`, {
            method: 'DELETE',
        });
    },

    async getFollowers(userId) {
        return this.request(`/users/${userId}/followers`);
    },

    async getFollowing(userId) {
        return this.request(`/users/${userId}/following`);
    },

    // Feed
    async getFeed(type = 'all', limit = 20, offset = 0) {
        return this.request(`/feed?type=${type}&limit=${limit}&offset=${offset}`);
    },

};



export function isLoggedIn() {
    return !!localStorage.getItem('token');
}

export function logout() {
    localStorage.removeItem('token');
    window.location.href = 'index.html';
}

export function updateNavigation() {
    const loggedIn = isLoggedIn();
    const loginLink = document.getElementById('loginLink');
    const profileLink = document.getElementById('profileLink');
    const logoutBtn = document.getElementById('logoutBtn');

    if (loginLink) loginLink.classList.toggle('hidden', loggedIn);
    if (profileLink) profileLink.classList.toggle('hidden', !loggedIn);
    if (logoutBtn) {
        logoutBtn.classList.toggle('hidden', !loggedIn);
        logoutBtn.addEventListener('click', logout);
    }
}