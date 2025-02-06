let currentQuestion = null;
let startTime = null;
let gameTimer = null;
let totalTime = 0;
let questionsAnswered = 0;
let gameTimeLimit = 20000; // 20 seconds in milliseconds
let isGameActive = false;

async function logout() {
    try {
        const response = await fetch('/api/logout', {
            method: 'POST'
        });
        if (response.ok) {
            window.location.href = '/';
        }
    } catch (error) {
        console.error('Logout failed:', error);
    }
}

function displayUsername() {
    const usernameDisplay = document.getElementById('username-display');
    if (usernameDisplay) {
        fetch('/api/user')
            .then(response => response.json())
            .then(data => {
                usernameDisplay.textContent = `Welcome, ${data.username}!`;
            })
            .catch(console.error);
    }
}

if (document.getElementById('game-section')) {
    displayUsername();
}

function startNewGame() {
    questionsAnswered = 0;
    totalTime = 0;
    isGameActive = true;
    updateStats();
    loadNewQuestion();
    startGameTimer();
}

function startGameTimer() {
    const startTimestamp = Date.now();
    const timerDisplay = document.getElementById('timer');
    
    if (gameTimer) clearInterval(gameTimer);
    
    gameTimer = setInterval(() => {
        const elapsed = Date.now() - startTimestamp;
        const remaining = (gameTimeLimit - elapsed) / 1000;
        
        if (remaining <= 0) {
            endGame();
        } else {
            timerDisplay.textContent = `Time: ${remaining.toFixed(1)}s`;
            if (remaining <= 3) {
                timerDisplay.style.color = '#ef4444';
            }
        }
    }, 100);
}

async function endGame() {
    isGameActive = false;
    clearInterval(gameTimer);
    
    const finalScore = questionsAnswered;
    const avgSpeed = questionsAnswered > 0 ? (totalTime / questionsAnswered / 1000) : 0;
    
    // Send final score to server
    await fetch('/api/leaderboard/update', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            score: finalScore,
            avgSpeed: avgSpeed
        })
    });

    // Show game over message
    const questionDisplay = document.getElementById('question-display');
    questionDisplay.textContent = `Game Over! Score: ${finalScore}`;
    questionDisplay.style.color = '#4ade80';
    
    // Disable answer input
    document.getElementById('answer').disabled = true;
    
    // Update leaderboard
    showLeaderboard('score');
}

async function loadNewQuestion() {
    if (!isGameActive) return;
    
    try {
        const response = await fetch('/api/questions/random', {
            credentials: 'same-origin'
        });
        
        if (!response.ok) {
            throw new Error('Failed to load question');
        }
        
        currentQuestion = await response.json();
        document.getElementById('question-display').textContent = currentQuestion.question;
        
        // Display fact if available
        if (currentQuestion.fact) {
            showFact(currentQuestion.fact);
        }
        
        document.getElementById('answer').value = '';
        startTime = Date.now();
    } catch (error) {
        console.error('Error loading question:', error);
        endGame();
    }
}

async function submitAnswer() {
    if (!isGameActive) return;
    
    const timeSpent = Date.now() - startTime;
    const userAnswer = parseInt(document.getElementById('answer').value);
    
    const response = await fetch('/api/questions/verify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            questionId: currentQuestion.id,
            answer: userAnswer,
            timeSpent: timeSpent
        }),
    });

    const result = await response.json();
    
    if (result.correct) {
        questionsAnswered++;
        totalTime += timeSpent;
        showFeedback(true);
        if (result.fact) {
            showFact(result.fact);
        }
    } else {
        showFeedback(false);
    }
    
    updateStats();
    loadNewQuestion();
}

function showFeedback(correct) {
    const display = document.getElementById('question-display');
    display.className = correct ? 'question correct' : 'question incorrect';
    setTimeout(() => {
        display.className = 'question';
    }, 500);
}

function updateStats() {
    const avgSpeed = questionsAnswered > 0 ? (totalTime / questionsAnswered / 1000) : 0;
    document.getElementById('score').textContent = `Score: ${questionsAnswered}`;
    document.getElementById('avg-speed').textContent = `Avg Speed: ${avgSpeed.toFixed(1)}s`;
}

// Add this function to check if we're on the leaderboard page
function isLeaderboardPage() {
    return window.location.pathname === '/leaderboard-page';
}

// Modify the showLeaderboard function
async function showLeaderboard(type) {
    const response = await fetch('/api/leaderboard?type=' + type);
    const leaderboard = await response.json();
    
    const leaderboardList = document.getElementById('leaderboard-list');
    leaderboardList.innerHTML = '';
    
    leaderboard.forEach((entry, index) => {
        const div = document.createElement('div');
        div.className = 'leaderboard-entry';
        div.innerHTML = `
            <div class="user-info">
                ${entry.avatar ? `<img src="${entry.avatar}" alt="avatar" class="avatar">` : ''}
                <span class="rank">#${index + 1} ${entry.username}</span>
            </div>
            <span class="stats">Score: ${entry.score} | Avg Speed: ${entry.avgSpeed.toFixed(1)}s</span>
        `;
        leaderboardList.appendChild(div);
    });

    // Update active tab
    if (isLeaderboardPage()) {
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.classList.remove('active');
        });
        document.querySelector(`[onclick="showLeaderboard('${type}')"]`).classList.add('active');
    }
}

// Initialize leaderboard if we're on the leaderboard page
if (isLeaderboardPage()) {
    showLeaderboard('score');
}

// Handle Enter key for answer submission
document.getElementById('answer').addEventListener('keypress', function(e) {
    if (e.key === 'Enter') {
        submitAnswer();
    }
});

// Update leaderboard periodically
setInterval(() => showLeaderboard('score'), 30000);

// Add new function to show facts
function showFact(fact) {
    // Remove any existing fact display
    const existingFact = document.querySelector('.fact-display');
    if (existingFact) {
        existingFact.remove();
    }

    const factDisplay = document.createElement('div');
    factDisplay.className = 'fact-display';
    factDisplay.textContent = fact;
    document.querySelector('.question-area').appendChild(factDisplay);

    // Animate the fact
    factDisplay.style.opacity = '0';
    factDisplay.style.transform = 'translateY(20px)';
    
    // Trigger animation
    setTimeout(() => {
        factDisplay.style.opacity = '1';
        factDisplay.style.transform = 'translateY(0)';
    }, 100);

    // Remove fact after 5 seconds
    setTimeout(() => {
        if (factDisplay.parentNode) {
            factDisplay.style.opacity = '0';
            factDisplay.style.transform = 'translateY(-20px)';
            setTimeout(() => factDisplay.remove(), 300);
        }
    }, 5000);
} 