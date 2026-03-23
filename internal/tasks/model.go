package tasks

import (
	"fmt"
	"strings"

	gtasks "google.golang.org/api/tasks/v1"
)

const resourceScheme = "gtasks:///"

type Task struct {
	ID         string `json:"id"`
	URI        string `json:"uri"`
	Title      string `json:"title"`
	Notes      string `json:"notes,omitempty"`
	Status     string `json:"status,omitempty"`
	Due        string `json:"due,omitempty"`
	Updated    string `json:"updated,omitempty"`
	Recurrence string `json:"recurrence,omitempty"`
}

func ResourceURI(id string) string {
	return resourceScheme + id
}

func ParseResourceURI(uri string) (string, error) {
	if !strings.HasPrefix(uri, resourceScheme) {
		return "", fmt.Errorf("unsupported resource URI")
	}

	id := strings.TrimPrefix(uri, resourceScheme)
	if id == "" {
		return "", fmt.Errorf("missing task ID in resource URI")
	}

	return id, nil
}

func fromGoogleTask(task *gtasks.Task) Task {
	if task == nil {
		return Task{}
	}

	notes, recurrence := splitNotesAndRecurrence(task.Notes)

	return Task{
		ID:         task.Id,
		URI:        ResourceURI(task.Id),
		Title:      task.Title,
		Notes:      notes,
		Status:     task.Status,
		Due:        task.Due,
		Updated:    task.Updated,
		Recurrence: recurrence,
	}
}
