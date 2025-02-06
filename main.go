package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"mental-math-app/db"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// Structures for our application
type User struct {
	ID       int     `json:"id"`
	Username string  `json:"username"`
	Password string  `json:"password"`
	Score    int     `json:"score"`
	AvgSpeed float64 `json:"avgSpeed"`
}

type Question struct {
	ID        int    `json:"id"`
	Question  string `json:"question"`
	Answer    int    `json:"answer"`
	Level     string `json:"level"`
	Fact      string `json:"fact,omitempty"` // For number facts
	Timestamp int64  `json:"timestamp"`
}

type LeaderboardEntry struct {
	Username string  `json:"username"`
	Score    int     `json:"score"`
	AvgSpeed float64 `json:"avgSpeed"`
}

type GameResult struct {
	Score    int     `json:"score"`
	AvgSpeed float64 `json:"avgSpeed"`
}

var (
	store             = sessions.NewCookieStore([]byte(getSessionKey()))
	githubOauthConfig = &oauth2.Config{
		ClientID:     "your-client-id",
		ClientSecret: "your-client-secret",
		RedirectURL:  "http://localhost:8080/auth/github/callback",
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}
	currentQuestion Question
)

func init() {
	db.InitDB()

	// Configure session
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	r := mux.NewRouter()

	// Add security middleware
	r.Use(securityMiddleware)

	// Routes
	r.HandleFunc("/api/register", RegisterHandler).Methods("POST")
	r.HandleFunc("/api/login", LoginHandler).Methods("POST")
	r.HandleFunc("/api/questions/random", requireAuth(GetRandomQuestion)).Methods("GET")
	r.HandleFunc("/api/questions/verify", requireAuth(VerifyAnswer)).Methods("POST")
	r.HandleFunc("/api/leaderboard", requireAuth(GetLeaderboard)).Methods("GET")
	r.HandleFunc("/leaderboard-page", requireAuth(LeaderboardPageHandler)).Methods("GET")
	r.HandleFunc("/api/leaderboard/update", requireAuth(UpdateScore)).Methods("POST")
	r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/register", RegisterPageHandler)
	r.HandleFunc("/game", requireAuth(GamePageHandler))
	r.HandleFunc("/api/logout", LogoutHandler).Methods("POST")
	r.HandleFunc("/api/user", requireAuth(GetCurrentUser)).Methods("GET")
	r.HandleFunc("/auth/github/login", GithubLoginHandler)
	r.HandleFunc("/auth/github/callback", GithubCallbackHandler)

	// Serve static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Printf("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/index.html")
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate input
	if user.Username == "" || user.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	err := db.RegisterUser(user.Username, user.Password)
	if err != nil {
		if err.Error() == "username already exists" {
			http.Error(w, "Username already exists", http.StatusConflict)
		} else {
			http.Error(w, "Registration failed", http.StatusInternalServerError)
			log.Printf("Registration error: %v", err)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate input
	if user.Username == "" || user.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	userID, err := db.ValidateUser(user.Username, user.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	session.Values["userID"] = userID
	session.Values["username"] = user.Username

	if err := session.Save(r, w); err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func GetRandomQuestion(w http.ResponseWriter, r *http.Request) {
	currentQuestion = generateQuestion()
	json.NewEncoder(w).Encode(currentQuestion)
}

func VerifyAnswer(w http.ResponseWriter, r *http.Request) {
	var submission struct {
		QuestionID int   `json:"questionId"`
		Answer     int   `json:"answer"`
		TimeSpent  int64 `json:"timeSpent"`
	}

	if err := json.NewDecoder(r.Body).Decode(&submission); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get fact for correct answer
	fact := ""
	if submission.Answer == currentQuestion.Answer {
		if f, err := getNumberFact(submission.Answer); err == nil {
			fact = f
		}
	}

	response := map[string]interface{}{
		"correct": submission.Answer == currentQuestion.Answer,
		"fact":    fact,
	}
	json.NewEncoder(w).Encode(response)
}

func GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	sortBy := r.URL.Query().Get("type")

	leaderboard, err := db.GetLeaderboard(sortBy)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(leaderboard)
}

func LeaderboardPageHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/leaderboard.html")
}

func UpdateScore(w http.ResponseWriter, r *http.Request) {
	var result GameResult
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	session, _ := store.Get(r, "session-name")
	userID, ok := session.Values["userID"].(int)
	if !ok {
		http.Error(w, "Not logged in", http.StatusUnauthorized)
		return
	}

	err := db.SaveScore(userID, result.Score, result.AvgSpeed)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func generateQuestion() Question {
	num1 := rand.Intn(90) + 10
	num2 := rand.Intn(90) + 10
	operations := []string{"+", "-", "×"}
	op := operations[rand.Intn(len(operations))]

	var answer int
	switch op {
	case "+":
		answer = num1 + num2
	case "-":
		answer = num1 - num2
	case "×":
		num2 = rand.Intn(9) + 2 // Make multiplication easier (2-10)
		answer = num1 * num2
	}

	// Get a fun fact about one of the numbers
	fact := ""
	if rand.Float32() < 0.5 { // 50% chance to get a fact
		if f, err := getNumberFact(num1); err == nil {
			fact = f
		}
	} else {
		if f, err := getNumberFact(num2); err == nil {
			fact = f
		}
	}

	return Question{
		ID:        rand.Int(),
		Question:  strconv.Itoa(num1) + " " + op + " " + strconv.Itoa(num2),
		Answer:    answer,
		Level:     "medium",
		Fact:      fact,
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
	}
}

func RegisterPageHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/register.html")
}

func GamePageHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/game.html")
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	session.Values = map[interface{}]interface{}{}
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	username, ok := session.Values["username"].(string)
	if !ok {
		http.Error(w, "Not logged in", http.StatusUnauthorized)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"username": username,
	})
}

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session-name")
		if _, ok := session.Values["userID"].(int); !ok {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func getSessionKey() string {
	key := os.Getenv("SESSION_KEY")
	if key == "" {
		key = "your-default-secret-key-change-this-in-production"
	}
	return key
}

func securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		next.ServeHTTP(w, r)
	})
}

func GithubLoginHandler(w http.ResponseWriter, r *http.Request) {
	url := githubOauthConfig.AuthCodeURL("state")
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func GithubCallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	token, err := githubOauthConfig.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "Failed to get token", http.StatusInternalServerError)
		return
	}

	client := githubOauthConfig.Client(r.Context(), token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var githubUser struct {
		Login string `json:"login"`
		ID    int64  `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
		http.Error(w, "Failed to decode user info", http.StatusInternalServerError)
		return
	}

	// Create or get user
	userID, err := db.GetOrCreateGithubUser(githubUser.ID, githubUser.Login)
	if err != nil {
		http.Error(w, "Failed to process user", http.StatusInternalServerError)
		return
	}

	session, _ := store.Get(r, "session-name")
	session.Values["userID"] = userID
	session.Values["username"] = githubUser.Login
	session.Save(r, w)

	http.Redirect(w, r, "/game", http.StatusTemporaryRedirect)
}

func getNumberFact(number int) (string, error) {
	// Randomly choose fact type
	factTypes := []string{"math", "trivia", "year"}
	factType := factTypes[rand.Intn(len(factTypes))]

	resp, err := http.Get(fmt.Sprintf("http://numbersapi.com/%d/%s", number, factType))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func getRandomAvatar() (string, error) {
	resp, err := http.Get("https://randomuser.me/api/?inc=picture")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Results []struct {
			Picture struct {
				Thumbnail string `json:"thumbnail"`
			} `json:"picture"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Results) == 0 {
		return "", fmt.Errorf("no avatar received")
	}

	return result.Results[0].Picture.Thumbnail, nil
}
