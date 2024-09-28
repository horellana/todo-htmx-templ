package main

import (
	"os"
	"fmt"
	"time"
	"context"
	"strconv"
	"log/slog"
	"net/http"
	schema "github.com/gorilla/schema"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	sqlbuilder "github.com/Masterminds/squirrel"
)

const INPUT_PLACEHOLDER = "Try 'Buying Milk'"

type Todo struct {
	Id int `db:"id"`
	Name string `db:"name"`
	Completed bool `db:"completed"`
	CompletedAt *string `db:"completedAt"`
	CreatedAt string `db:"createdAt"`
	UpdatedAt string `db:"updatedAt"`
	DeletedAt *string `db:"deletedAt"`
}

type UpdateTodoPayload struct {
	Completed bool `schema:completed`
}

type CreateTodoPayload struct {
	Name string `schema:name`
}

func ConnectToDatabase(connectionString string) (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite3", connectionString)

	if err != nil {
		return nil, err
	}

	return db, nil
}

func CountTodos(db *sqlx.DB) (int, error) {
	queryBuilder := sqlbuilder.
		Select("COUNT(*)").
		From("todos").
		Where("deletedAt is null")

	query, parameters, err := queryBuilder.ToSql()

	if err != nil {
		return 0, err
	}

	count := 0
	err = db.Get(&count, query, parameters...)

	return count, err
}

func ListTodos(db *sqlx.DB) ([]Todo, error) {
	queryBuilder := sqlbuilder.
		Select("*").
		From("todos").
		Where("deletedAt is null").
		OrderBy("createdAt ASC")

	query, parameters, err := queryBuilder.ToSql()

	if err != nil {
		return []Todo{}, err
	}

	fmt.Println(query)

	todos := []Todo{}
	err = db.Select(&todos, query, parameters...)

	return todos, err
}

func RemoveTodo(db *sqlx.DB, todoId int) (int, error) {
	now := time.Now().Format("2006-01-02 15:04:05")

	queryBuilder := sqlbuilder.
		Update("todos").
		Where("id = ?", todoId).
		Set("updatedAt", now).
		Set("deletedAt", now)

	query, parameters, err := queryBuilder.ToSql()

	if err != nil {
		return 0, err
	}

	_, err = db.Exec(query, parameters...)

	if err != nil {
		return 0, err
	}

	count, countErr := CountTodos(db)

	return count, countErr
}

func UpdateTodo(db *sqlx.DB, todoId int, completed bool) (Todo, error) {
	now := time.Now().Format("2006-01-02 15:04:05")

	queryBuilder := sqlbuilder.
		Update("todos").
		Where("id = ?", todoId).
		Set("updatedAt", now).
		Set("completed", completed).
		Suffix("returning id, name, completed, completedAt, createdAt, updatedAt, deletedAt")

	if (completed) {
		queryBuilder = queryBuilder.Set("completedAt", now)
	} else {
		queryBuilder = queryBuilder.Set("completedAt", nil)
	}

	query, parameters, err := queryBuilder.ToSql()

	if err != nil {
		return Todo{}, err
	}

	todo := Todo{}
	err = db.Get(&todo, query, parameters...)

	if err != nil {
		return Todo{}, err
	}

	return todo, nil
}

func CreateTodo(db *sqlx.DB, name string) (Todo, error) {
	queryBuilder := sqlbuilder.
		Insert("todos").
		Columns("name", "completed").
		Values(name, 0).
		Suffix("returning id, name, completed, completedAt, createdAt, updatedAt, deletedAt")

	query, parameters, err := queryBuilder.ToSql()

	if err != nil {
		return Todo{}, err
	}

	todo := Todo{}

	err = db.Get(&todo, query, parameters...)

	slog.Debug("CREATE_TODO", "MESSAGE", query)
	slog.Debug("CREATE_TODO", "MESSAGE", parameters)

	if err != nil {
		return Todo{}, err
	}

	return todo, err
}

func RootHandler(db *sqlx.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")

		todos, err := ListTodos(db)

		if err != nil {
			slog.Error("ROOT_HANDLER", "ERROR", err)
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}

		component := Index(todos, INPUT_PLACEHOLDER, "")

		component.Render(context.Background(), w)
	})
}

func UpdateTodoHandler(db *sqlx.DB, decoder *schema.Decoder) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")

		err :=  r.ParseForm()

		if err != nil {
			message := "Could not parse payload"
			slog.Error("UPDATE_TODO", "MESSAGE", message)
			http.Error(w, message, http.StatusBadRequest)
			return
		}

		payload := new(UpdateTodoPayload)
		err = decoder.Decode(payload, r.Form)

		if err != nil {
			message := "Could not decode payload"
			slog.Error("UPDATE_TODO", "MESSAGE", message)
			http.Error(w, message, http.StatusBadRequest)
			return
		}

		todoId, todoIdErr := strconv.Atoi(r.PathValue("id"))

		if todoIdErr != nil {
			message := fmt.Sprintf("Bad todo id: %s", r.PathValue("id"))
			slog.Error("UPDATE_TODO", "MESSAGE", message)
			http.Error(w, message, http.StatusBadRequest)
			return
		}

		slog.Debug("UPDATE_TODO", "REQUEST_PAYLOAD", payload)

		newTodo, newTodoErr := UpdateTodo(db, todoId, payload.Completed)

		if newTodoErr != nil {
			slog.Error("ROOT_HANDLER", "ERROR", newTodoErr)
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}

		component := TodoRow(newTodo)

		component.Render(context.Background(), w)
	})
}

func CreateTodoHandler(db *sqlx.DB, decoder *schema.Decoder) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err :=  r.ParseForm()

		if err != nil {
			message := "Could not parse payload"
			slog.Error("CREATE_TODO", "MESSAGE", message)
			NewTodoErrorOOB(INPUT_PLACEHOLDER, message).Render(context.Background(), w)
			return
		}

		payload := new(CreateTodoPayload)
		err = decoder.Decode(payload, r.Form)

		if err != nil {
			message := "Could not decode payload"
			slog.Error("CREATE_TODO", "MESSAGE", message)
			NewTodoErrorOOB(INPUT_PLACEHOLDER, message).Render(context.Background(), w)
			return
		}

		if len(payload.Name) < 1 {
			slog.Error("CREATE_TODO", "MESSAGE", "TODO can not be empty")
			NewTodoErrorOOB(INPUT_PLACEHOLDER, "TODO can not be empty").Render(context.Background(), w)
			return
		}

		todo, todoErr := CreateTodo(db, payload.Name)

		if todoErr != nil {
			slog.Error("CREATE_TODO", "ERROR", todoErr)
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}

		slog.Debug("CREATE_TODO", "MESSAGE", todo)

		component := NewTodoOOB(INPUT_PLACEHOLDER, todo)

		w.Header().Add("Content-Type", "text/html")
		component.Render(context.Background(), w)
	})
}

func DeleteTodoHandler(db *sqlx.DB, decoder *schema.Decoder) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		todoId, todoIdErr := strconv.Atoi(r.PathValue("id"))

		if todoIdErr != nil {
			message := fmt.Sprintf("Bad todo id: %s", r.PathValue("id"))
			slog.Error("DELETE_TODO", "MESSAGE", message)
			http.Error(w, message, http.StatusBadRequest)
			return
		}

		todosCount, err := RemoveTodo(db, todoId)

		if err != nil {
			slog.Error("DELETE_TODO_HANDLER", "ERROR", err)
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}

		slog.Error("DELETE_TODO", "TODO_ID", todoId)

		w.Header().Add("Content-Type", "text/html")
		RemoveTodoOOB(todoId, todosCount).Render(context.Background(), w)
	})
}

func main() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	decoder := schema.NewDecoder()

	server := http.NewServeMux()
	db, err := ConnectToDatabase("todos.db")

	if err != nil {
		slog.Error("CONNECT_DATABASE", "MESSAGE", err)
		return
	}

	staticFilesHandler := http.FileServer(http.Dir("./static"))

	server.Handle("/static/", http.StripPrefix("/static", staticFilesHandler))

	server.Handle("GET /todos", RootHandler(db))
	server.Handle("POST /todos", CreateTodoHandler(db, decoder))
	server.Handle("PUT /todos/{id}", UpdateTodoHandler(db, decoder))
	server.Handle("DELETE /todos/{id}", DeleteTodoHandler(db, decoder))

	port := os.Getenv("PORT")

	if len(port) < 1 {
		port = "3000"
	}

	listenAddress := fmt.Sprintf("0.0.0.0:%s", port)

	slog.Debug("MAIN", "LISTEN ADDRESS", listenAddress)

	http.ListenAndServe(listenAddress, server)
}
