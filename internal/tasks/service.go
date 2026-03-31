package tasks

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gtasks-mcp/internal/logging"

	"google.golang.org/api/option"
	gtasks "google.golang.org/api/tasks/v1"
)

type Service struct {
	svc        *gtasks.Service
	taskListID string
	initErr    error
}

func New(client *http.Client, listName string) (*Service, error) {
	logging.Info("initializing google tasks service", "task_list", listName)
	svc, err := gtasks.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	s := &Service{svc: svc}
	s.taskListID, err = s.ensureList(listName)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func NewUnavailable(listName string, err error) *Service {
	return &Service{
		taskListID: listName,
		initErr:    fmt.Errorf("google tasks service unavailable: %w", err),
	}
}

func (s *Service) TaskListID() string {
	return s.taskListID
}

func (s *Service) ensureReady() error {
	if s == nil {
		return errors.New("google tasks service unavailable")
	}
	if s.initErr != nil {
		return s.initErr
	}
	if s.svc == nil {
		return errors.New("google tasks service unavailable")
	}
	return nil
}

func (s *Service) ensureList(name string) (string, error) {
	logging.Debug("ensuring task list exists", "task_list", name)
	lists, err := s.svc.Tasklists.List().Do()
	if err != nil {
		return "", err
	}

	for _, l := range lists.Items {
		if l.Title == name {
			logging.Info("using existing task list", "task_list", name, "task_list_id", l.Id)
			return l.Id, nil
		}
	}

	l, err := s.svc.Tasklists.Insert(&gtasks.TaskList{Title: name}).Do()
	if err != nil {
		return "", err
	}

	logging.Info("created task list", "task_list", name, "task_list_id", l.Id)
	return l.Id, nil
}

func retry(fn func() error) error {
	var err error
	for i := 0; i < 3; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		logging.Warn("retrying google tasks call", "attempt", i+1, "error", err)
		time.Sleep(time.Duration(1<<i) * time.Second)
	}
	return err
}

func (s *Service) List() ([]Task, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	logging.Debug("listing tasks", "task_list_id", s.taskListID)
	var res *gtasks.Tasks
	err := retry(func() error {
		var err error
		res, err = s.svc.Tasks.List(s.taskListID).Do()
		return err
	})
	if err != nil {
		return nil, err
	}

	out := make([]Task, 0, len(res.Items))
	for _, task := range res.Items {
		out = append(out, fromGoogleTask(task))
	}

	return out, nil
}

func (s *Service) Search(q string) ([]Task, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	logging.Debug("searching tasks", "task_list_id", s.taskListID, "query", q)
	items, err := s.List()
	if err != nil {
		return nil, err
	}

	var out []Task
	for _, t := range items {
		if strings.Contains(strings.ToLower(t.Title), strings.ToLower(q)) ||
			strings.Contains(strings.ToLower(t.Notes), strings.ToLower(q)) {
			out = append(out, t)
		}
	}
	return out, nil
}

func (s *Service) Create(title, notes, due, recurrence string) (Task, error) {
	if err := s.ensureReady(); err != nil {
		return Task{}, err
	}
	recurrence = normalizeRecurrence(recurrence)
	if !validRecurrence(recurrence) {
		return Task{}, fmt.Errorf("unsupported recurrence %q", recurrence)
	}
	if recurrence != "" && strings.TrimSpace(due) == "" {
		return Task{}, fmt.Errorf("due is required for recurring tasks")
	}

	logging.Info("creating task", "task_list_id", s.taskListID, "title", title, "has_due", due != "")
	var task *gtasks.Task
	err := retry(func() error {
		var err error
		task, err = s.svc.Tasks.Insert(s.taskListID, &gtasks.Task{
			Title: title,
			Notes: composeNotes(notes, recurrence),
			Due:   due,
		}).Do()
		return err
	})
	return fromGoogleTask(task), err
}

func (s *Service) Update(id string, title, notes, status, due, recurrence *string) (Task, error) {
	if err := s.ensureReady(); err != nil {
		return Task{}, err
	}
	logging.Info("updating task", "task_list_id", s.taskListID, "task_id", id, "status_supplied", status != nil, "due_supplied", due != nil, "recurrence_supplied", recurrence != nil)
	existing, err := s.Get(id)
	if err != nil {
		return Task{}, err
	}

	nextTitle := existing.Title
	if title != nil {
		nextTitle = *title
	}

	nextNotes := existing.Notes
	if notes != nil {
		nextNotes = *notes
	}

	nextStatus := existing.Status
	if status != nil {
		nextStatus = *status
	}

	nextDue := existing.Due
	if due != nil {
		nextDue = *due
	}

	nextRecurrence := existing.Recurrence
	if recurrence != nil {
		nextRecurrence = normalizeRecurrence(*recurrence)
	}
	if !validRecurrence(nextRecurrence) {
		return Task{}, fmt.Errorf("unsupported recurrence %q", nextRecurrence)
	}
	if nextRecurrence != "" && strings.TrimSpace(nextDue) == "" {
		return Task{}, fmt.Errorf("due is required for recurring tasks")
	}

	var task *gtasks.Task
	err = retry(func() error {
		var err error
		task, err = s.svc.Tasks.Patch(s.taskListID, id, &gtasks.Task{
			Id:     id,
			Title:  nextTitle,
			Notes:  composeNotes(nextNotes, nextRecurrence),
			Status: nextStatus,
			Due:    nextDue,
		}).Do()
		return err
	})
	if err != nil {
		return Task{}, err
	}

	updated := fromGoogleTask(task)
	if shouldRollRecurringTask(existing, updated) {
		if _, err := s.createNextRecurringTask(updated); err != nil {
			return Task{}, err
		}
	}

	return updated, nil
}

func (s *Service) Delete(id string) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	logging.Info("deleting task", "task_list_id", s.taskListID, "task_id", id)
	return retry(func() error {
		return s.svc.Tasks.Delete(s.taskListID, id).Do()
	})
}

func (s *Service) Clear() error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	logging.Info("clearing completed tasks", "task_list_id", s.taskListID)
	return retry(func() error {
		return s.svc.Tasks.Clear(s.taskListID).Do()
	})
}

func (s *Service) Get(id string) (Task, error) {
	if err := s.ensureReady(); err != nil {
		return Task{}, err
	}
	logging.Debug("reading task", "task_list_id", s.taskListID, "task_id", id)
	var t *gtasks.Task
	err := retry(func() error {
		var err error
		t, err = s.svc.Tasks.Get(s.taskListID, id).Do()
		return err
	})
	return fromGoogleTask(t), err
}

func shouldRollRecurringTask(previous, updated Task) bool {
	return previous.Status != "completed" && updated.Status == "completed" && updated.Recurrence != ""
}

func (s *Service) createNextRecurringTask(task Task) (Task, error) {
	nextDue, err := nextDueDate(task.Due, task.Recurrence, time.Now())
	if err != nil {
		return Task{}, err
	}

	logging.Info("creating next recurring task occurrence", "task_list_id", s.taskListID, "source_task_id", task.ID, "recurrence", task.Recurrence, "next_due", nextDue)
	return s.Create(task.Title, task.Notes, nextDue, task.Recurrence)
}
