package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	_ "github.com/lib/pq"
)

type Server struct {
	db *sql.DB
}

func main() {
	connStr := "postgresql://Mona:42@database:5432/tasks?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	server := &Server{db: db}

	err = initializeDatabase(db)

	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("ToDo list")
	})
	mux.HandleFunc("GET /list", checkAuth(server.getToDoList))
	mux.HandleFunc("POST /task", checkAuth(server.postTask))
	mux.HandleFunc("PUT /task", checkAuth(server.editTask))
	mux.HandleFunc("DELETE /task", checkAuth(server.deleteTask))
	err2 := http.ListenAndServe(":8080", mux)
	if err2 != nil {
		fmt.Println("Error happened", err2.Error())
		return
	}
}

func initializeDatabase(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id SERIAL PRIMARY KEY,
			description TEXT NOT NULL
		)
	`)
	return err
}

type Authorisation struct {
	UserName string
	Password string
}

var User1 = Authorisation{
	UserName: "Mona",
	Password: "42",
}

var User2 = Authorisation{
	UserName: "Liza",
	Password: "315",
}

func checkAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if (username != User1.UserName || password != User1.Password) && (username != User2.UserName || password != User2.Password) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

type TaskManager struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
}

// var tasks = []TaskManager{
// 	{ID: 1, Description: "Open computer"},
// 	{ID: 2, Description: "Do homework"},
// 	{ID: 3, Description: "Close computer"},
// }

func (s *Server) getToDoList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query("SELECT id, description FROM tasks")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tasks []TaskManager
	for rows.Next() {
		var task TaskManager
		if err := rows.Scan(&task.ID, &task.Description); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(tasks)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s *Server) postTask(w http.ResponseWriter, r *http.Request) {
	var newTask TaskManager
	err := json.NewDecoder(r.Body).Decode(&newTask)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = s.db.QueryRow("INSERT INTO tasks (description) VALUES ($1) RETURNING id", newTask.Description).Scan(&newTask.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(newTask)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s *Server) editTask(w http.ResponseWriter, r *http.Request) {
	var updatedTask TaskManager
	err := json.NewDecoder(r.Body).Decode(&updatedTask)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err = s.db.Exec("UPDATE tasks SET description=$1 WHERE id=$2", updatedTask.Description, updatedTask.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(updatedTask)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err = s.db.Exec("DELETE FROM tasks WHERE id=$1", id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
