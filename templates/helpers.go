package templates

import (
	"fmt"
	. "todo.app/models"
)

func GetTodoRowClass(todo Todo) string {
	result := "border-2 p-4 mb-1 hover:bg-green-100"

	if (todo.CompletedAt != nil && *todo.CompletedAt != "") {
		result = fmt.Sprintf("%s border-green-500", result)
	}

	return result
}
