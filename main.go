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
)

const INPUT_PLACEHOLDER = "Try 'Buying Milk'"

type Todo struct {
	Id int
	Name string
	Completed bool
	CompletedAt string
}

var TODOS = []Todo{
	{
		Id: 1,
		Name: "Todo 1",
		Completed: false,
	},
	{
		Id: 2,
		Name: "Todo 2",
		Completed: true,
		CompletedAt: time.Now().Format("2006-01-02 15:04:05"),
	},
}

type UpdateTodoPayload struct {
	Completed bool `schema:completed`
}

type CreateTodoPayload struct {
	Name string `schema:name`
}

func ListTodos() []Todo {
	return TODOS
}

func RemoveTodo(todoId int, todos []Todo) []Todo {
	result := []Todo{}

	for _, todo := range todos {
		if todo.Id != todoId {
			result = append(result, todo)
		}
	}

	TODOS = result
	return result
}

func UpdateTodo(todoId int, completed bool) Todo {
	TODOS[todoId - 1].Completed = completed

	if (completed) {
		TODOS[todoId - 1].CompletedAt = time.Now().Format("2006-01-02 15:04:05")
	} else {
		TODOS[todoId - 1].CompletedAt = ""
	}

	return TODOS[todoId - 1]
}

func CreateTodo(name string) Todo {
	todo := Todo{
		Id: len(TODOS) + 1,
		Name: name,
		Completed: false,
	}

	TODOS = append(TODOS, todo)

	return todo
}

func RootHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")

		todos := ListTodos()
		component := Index(todos, INPUT_PLACEHOLDER, "")

		component.Render(context.Background(), w)
	})
}

func UpdateTodoHandler(decoder *schema.Decoder) http.Handler {
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

		newTodo := UpdateTodo(todoId, payload.Completed)

		component := TodoRow(newTodo)


		component.Render(context.Background(), w)
	})
}

func CreateTodoHandler(decoder *schema.Decoder) http.Handler {
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

		todo := CreateTodo(payload.Name)

		slog.Debug("CREATE_TODO", "MESSAGE", todo)

		component := NewTodoOOB(INPUT_PLACEHOLDER, todo)

		w.Header().Add("Content-Type", "text/html")
		component.Render(context.Background(), w)
	})
}

func DeleteTodoHandler(decoder *schema.Decoder) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		todoId, todoIdErr := strconv.Atoi(r.PathValue("id"))

		if todoIdErr != nil {
			message := fmt.Sprintf("Bad todo id: %s", r.PathValue("id"))
			slog.Error("DELETE_TODO", "MESSAGE", message)
			http.Error(w, message, http.StatusBadRequest)
			return
		}

		todos := RemoveTodo(todoId, TODOS)
		slog.Error("DELETE_TODO", "TODO_ID", todoId)

		w.Header().Add("Content-Type", "text/html")
		RemoveTodoOOB(todoId, todos).Render(context.Background(), w)
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

	staticFilesHandler := http.FileServer(http.Dir("./static"))

	server.Handle("/static/", http.StripPrefix("/static", staticFilesHandler))

	server.Handle("GET /todos", RootHandler())
	server.Handle("POST /todos", CreateTodoHandler(decoder))
	server.Handle("PUT /todos/{id}", UpdateTodoHandler(decoder))
	server.Handle("DELETE /todos/{id}", DeleteTodoHandler(decoder))

	port := os.Getenv("PORT")

	if len(port) < 1 {
		port = "3000"
	}

	listenAddress := fmt.Sprintf("0.0.0.0:%s", port)

	slog.Debug("MAIN", "LISTEN ADDRESS", listenAddress)

	http.ListenAndServe(listenAddress, server)
}
