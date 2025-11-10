import { api, isLoggedIn } from './api.js';

// Redirect if already logged in
if (isLoggedIn()) {
    window.location.href = 'feed.html';
}

// Login Form
const loginForm = document.getElementById('loginForm');
if (loginForm) {
    loginForm.addEventListener('submit', async (e) => {
        e.preventDefault();

        const email = document.getElementById('email').value;
        const password = document.getElementById('password').value;
        const errorDiv = document.getElementById('errorMessage');

        try {
            const response = await api.login(email, password);
            localStorage.setItem('token', response.token);
            window.location.href = 'feed.html';
        } catch (error) {
            errorDiv.textContent = 'Invalid email or password';
            errorDiv.classList.remove('hidden');
        }
    });
}

// Signup Form
const signupForm = document.getElementById('signupForm');
if (signupForm) {
    signupForm.addEventListener('submit', async (e) => {
        e.preventDefault();

        const username = document.getElementById('username').value;
        const email = document.getElementById('email').value;
        const password = document.getElementById('password').value;
        const errorDiv = document.getElementById('errorMessage');

        try {
            const response = await api.signup(username, email, password);
            localStorage.setItem('token', response.token);
            window.location.href = 'feed.html';
        } catch (error) {
            errorDiv.textContent = error.message || 'Signup failed';
            errorDiv.classList.remove('hidden');
        }
    });
}