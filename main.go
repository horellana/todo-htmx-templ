package main

import (
	"os"
	"fmt"
	"context"
	"strconv"
	"log/slog"
	"net/http"
	schema "github.com/gorilla/schema"
)

type Todo struct {
	Id int
	Name string
	Completed bool
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
	},
}

type UpdateTodoPayload struct {
	Completed bool `schema:completed`
}

type CreateTodoPayload struct {
	Name string `schema:name`
}

func RootHandler(w http.ResponseWriter, r *http.Request) {
	component := Index(TODOS)
	component.Render(context.Background(), w)
}

func UpdateTodoHandler(w http.ResponseWriter, r *http.Request) {
	err :=  r.ParseForm()

	if err != nil {
		message := "Could not parse payload"
		slog.Error("UPDATE_TODO", "MESSAGE", message)
		http.Error(w, message, http.StatusBadRequest)
		return
	}

	payload := new(UpdateTodoPayload)
	decoder := schema.NewDecoder()

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

	TODOS[todoId - 1].Completed = payload.Completed

	slog.Debug("UPDATE_TODO", "REQUEST_PAYLOAD", TODOS)

	component := TodoRow(TODOS[todoId - 1])
	component.Render(context.Background(), w)
}

func CreateTodoHandler(w http.ResponseWriter, r *http.Request) {
	err :=  r.ParseForm()

	if err != nil {
		message := "Could not parse payload"
		slog.Error("CREATE_TODO", "MESSAGE", message)
		http.Error(w, message, http.StatusBadRequest)
		return
	}

	payload := new(CreateTodoPayload)
	decoder := schema.NewDecoder()

	err = decoder.Decode(payload, r.Form)

	if err != nil {
		message := "Could not decode payload"
		slog.Error("CREATE_TODO", "MESSAGE", message)
		http.Error(w, message, http.StatusBadRequest)
		return
	}

	todo := Todo{
		Id: len(TODOS) + 1,
		Name: payload.Name,
		Completed: false,
	}

	slog.Debug("CREATE_TODO", "MESSAGE", todo)

	component := TodoRow(todo)

	TODOS = append(TODOS, todo)

	component.Render(context.Background(), w)
}

func main() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	server := http.NewServeMux()

	server.Handle("GET /", http.HandlerFunc(RootHandler))
	server.Handle("POST /", http.HandlerFunc(CreateTodoHandler))
	server.Handle("PUT /{id}", http.HandlerFunc(UpdateTodoHandler))

	http.ListenAndServe("0.0.0.0:3000", server)
}
