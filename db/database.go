package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var DB *sql.DB

func InitDB() {
	var err error
	DB, err = sql.Open("sqlite3", "./mental_math.db")
	if err != nil {
		log.Fatal(err)
	}

	// Create tables if they don't exist
	createTables()
}

func createTables() {
	// First, drop existing tables in correct order
	dropTables := []string{
		"DROP TABLE IF EXISTS scores;",
		"DROP TABLE IF EXISTS github_users;",
		"DROP TABLE IF EXISTS users;",
	}

	for _, query := range dropTables {
		_, err := DB.Exec(query)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Create users table with avatar column
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		avatar TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// Drop existing index if it exists
	DB.Exec("DROP INDEX IF EXISTS idx_username")

	// Create a unique index on username
	createUsernameIndex := `
	CREATE UNIQUE INDEX IF NOT EXISTS idx_username ON users(username);
	`

	createScoresTable := `
	CREATE TABLE IF NOT EXISTS scores (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		score INTEGER NOT NULL,
		avg_speed REAL NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users (id)
	);`

	createGithubUsersTable := `
	CREATE TABLE IF NOT EXISTS github_users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		github_id INTEGER UNIQUE NOT NULL,
		username TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// Execute all create statements
	_, err := DB.Exec(createUsersTable)
	if err != nil {
		log.Fatal(err)
	}

	_, err = DB.Exec(createUsernameIndex)
	if err != nil {
		log.Fatal(err)
	}

	_, err = DB.Exec(createScoresTable)
	if err != nil {
		log.Fatal(err)
	}

	_, err = DB.Exec(createGithubUsersTable)
	if err != nil {
		log.Fatal(err)
	}
}

func RegisterUser(username, password string) error {
	// First check if username already exists
	var exists bool
	err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", username).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("username already exists")
	}

	// If username doesn't exist, proceed with registration
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Get random avatar
	avatar := ""
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://randomuser.me/api/?inc=picture")
	if err == nil {
		defer resp.Body.Close()
		var result struct {
			Results []struct {
				Picture struct {
					Thumbnail string `json:"thumbnail"`
				} `json:"picture"`
			} `json:"results"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			if len(result.Results) > 0 {
				avatar = result.Results[0].Picture.Thumbnail
			}
		}
	}

	// Use a transaction to ensure data consistency
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT INTO users (username, password, avatar) VALUES (?, ?, ?)",
		username, string(hashedPassword), avatar)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func ValidateUser(username, password string) (int, error) {
	var id int
	var hashedPassword string

	err := DB.QueryRow("SELECT id, password FROM users WHERE username = ?",
		username).Scan(&id, &hashedPassword)
	if err != nil {
		return 0, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return 0, err
	}

	return id, nil
}

func SaveScore(userID int, score int, avgSpeed float64) error {
	_, err := DB.Exec(`
		INSERT INTO scores (user_id, score, avg_speed) 
		VALUES (?, ?, ?)`, userID, score, avgSpeed)
	return err
}

func GetLeaderboard(sortBy string) ([]LeaderboardEntry, error) {
	var query string
	if sortBy == "speed" {
		query = `
			SELECT u.username, s.score, s.avg_speed, u.avatar
			FROM scores s
			JOIN users u ON s.user_id = u.id
			ORDER BY s.avg_speed ASC
			LIMIT 10`
	} else {
		query = `
			SELECT u.username, s.score, s.avg_speed, u.avatar
			FROM scores s
			JOIN users u ON s.user_id = u.id
			ORDER BY s.score DESC
			LIMIT 10`
	}

	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	for rows.Next() {
		var entry LeaderboardEntry
		err := rows.Scan(&entry.Username, &entry.Score, &entry.AvgSpeed, &entry.Avatar)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func GetOrCreateGithubUser(githubID int64, username string) (int, error) {
	var userID int
	err := DB.QueryRow("SELECT id FROM github_users WHERE github_id = ?", githubID).Scan(&userID)
	if err == nil {
		return userID, nil
	}

	result, err := DB.Exec(`
		INSERT INTO github_users (github_id, username) 
		VALUES (?, ?)`, githubID, username)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	return int(id), err
}

type LeaderboardEntry struct {
	Username string  `json:"username"`
	Score    int     `json:"score"`
	AvgSpeed float64 `json:"avgSpeed"`
	Avatar   string  `json:"avatar"`
}
