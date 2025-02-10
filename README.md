# Mental Math Web Application

## Overview
The *Mental Math Web Application* is a Go-based web application designed to improve users' arithmetic skills through interactive math challenges. The project leverages the Go programming language to create a fast and efficient server, handling user interactions and scoring mechanisms.

## Features
- *User Authentication:* Allows users to sign up, log in, and track their progress.
- *Timed Math Challenges:* Users solve arithmetic problems within a time limit.
- *Difficulty Levels:* Supports multiple levels, from basic addition to complex problem-solving.
- *Leaderboards:* Tracks user scores and ranks them.
- *Real-time Feedback:* Immediate scoring and hints to help users improve.
- *API Support:* Exposes endpoints for frontend integration.

## Technologies Used
- *Go (Golang):* Backend server logic and API handling.
- *HTML, CSS, JavaScript:* Frontend UI components.
- *SQLite/PostgreSQL:* Database for storing user progress and scores.
- *RESTful API:* For communication between frontend and backend.
- *Docker (Optional):* Containerized deployment.

## Installation
### Prerequisites
- Install [Go](https://golang.org/doc/install)
- Install a database system (SQLite/PostgreSQL recommended)
- Install Git

### Steps to Run Locally
1. Clone the repository:
   sh
   git clone https://github.com/tarunkchauhan/go-lang-version2.git
   cd go-lang-version2
   
2. Install dependencies:
   sh
   go mod tidy
   
3. Set up environment variables (if required):
   sh
   export DB_URL="your_database_url"
   
4. Run the application:
   sh
   go run main.go
   
5. Open the browser and navigate to:
   sh
   http://localhost:8080
   

## API Endpoints
| Method | Endpoint           | Description                 |
|--------|-------------------|-----------------------------|
| GET    | /api/questions  | Fetches math problems      |
| POST   | /api/submit     | Submits user answers       |
| GET    | /api/leaderboard | Retrieves top scores      |

## Deployment
### Docker (Optional)
1. Build Docker image:
   sh
   docker build -t mental-math-app .
   
2. Run the container:
   sh
   docker run -p 8080:8080 mental-math-app
   

## Contribution Guidelines
1. Fork the repository.
2. Create a new branch:
   sh
   git checkout -b feature-name
   
3. Commit your changes:
   sh
   git commit -m "Added new feature"
   
4. Push and create a pull request.

## License
This project is open-source and available under the MIT License.

## Contact
For any issues or feature requests, feel free to reach out via GitHub: [Tarun K. Chauhan](https://github.com/tarunkchauhan).
