async function register() {
    const username = document.getElementById('username').value.trim();
    const password = document.getElementById('password').value;
    const confirmPassword = document.getElementById('confirm-password').value;
    
    // Clear previous errors
    clearErrors();
    
    // Validate inputs
    if (!username) {
        showError('username-error', 'Username is required');
        return;
    }
    
    if (!password) {
        showError('password-error', 'Password is required');
        return;
    }
    
    if (password.length < 6) {
        showError('password-error', 'Password must be at least 6 characters');
        return;
    }
    
    if (password !== confirmPassword) {
        showError('confirm-error', 'Passwords do not match');
        return;
    }

    try {
        const response = await fetch('/api/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password }),
        });

        if (response.ok) {
            alert('Registration successful! Please login.');
            window.location.href = '/';
        } else {
            const error = await response.text();
            if (error.includes('exists')) {
                showError('username-error', 'Username already exists');
            } else {
                showError('username-error', error);
            }
        }
    } catch (error) {
        showError('username-error', 'Registration failed. Please try again.');
    }
}

async function login() {
    const username = document.getElementById('username').value.trim();
    const password = document.getElementById('password').value;
    
    clearErrors();
    
    if (!username) {
        showError('username-error', 'Username is required');
        return;
    }
    
    if (!password) {
        showError('password-error', 'Password is required');
        return;
    }

    try {
        const response = await fetch('/api/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password }),
            credentials: 'same-origin' // Important for cookies
        });

        if (response.ok) {
            // Redirect to game page
            window.location.replace('/game');
        } else {
            const error = await response.text();
            showError('username-error', 'Invalid username or password');
            console.error('Login error:', error);
        }
    } catch (error) {
        showError('username-error', 'Login failed. Please try again.');
        console.error('Login error:', error);
    }
}

function showError(elementId, message) {
    const errorElement = document.getElementById(elementId);
    if (errorElement) {
        errorElement.textContent = message;
        errorElement.style.display = 'block';
    }
}

function clearErrors() {
    const errors = document.querySelectorAll('.error');
    errors.forEach(error => {
        error.textContent = '';
        error.style.display = 'none';
    });
} 