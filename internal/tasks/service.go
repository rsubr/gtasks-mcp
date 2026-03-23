package tasks

import (
	"context"
	"strings"
	"net/http"
	"time"

	gtasks "google.golang.org/api/tasks/v1"
	"google.golang.org/api/option"
)

type Service struct {
	svc *gtasks.Service
	taskListID string
}

func New(client *http.Client, listName string) *Service {
	svc, _ := gtasks.NewService(context.Background(), option.WithHTTPClient(client))
	s := &Service{svc: svc}
	s.taskListID = s.ensureList(listName)
	return s
}

func (s *Service) ensureList(name string) string {
	lists, _ := s.svc.Tasklists.List().Do()
	for _, l := range lists.Items {
		if l.Title == name {
			return l.Id
		}
	}
	l, _ := s.svc.Tasklists.Insert(&gtasks.TaskList{Title: name}).Do()
	return l.Id
}

func retry(fn func() error) error {
	var err error
	for i := 0; i < 3; i++ {
		err = fn()
		if err == nil { return nil }
		time.Sleep(time.Duration(1<<i) * time.Second)
	}
	return err
}

func (s *Service) List() ([]*gtasks.Task, error) {
	var res *gtasks.Tasks
	err := retry(func() error {
		var err error
		res, err = s.svc.Tasks.List(s.taskListID).Do()
		return err
	})
	if err != nil { return nil, err }
	return res.Items, nil
}

func (s *Service) Search(q string) ([]*gtasks.Task, error) {
	items, err := s.List()
	if err != nil { return nil, err }
	var out []*gtasks.Task
	for _, t := range items {
		if strings.Contains(strings.ToLower(t.Title), strings.ToLower(q)) ||
			strings.Contains(strings.ToLower(t.Notes), strings.ToLower(q)) {
			out = append(out, t)
		}
	}
	return out, nil
}

func (s *Service) Create(title, notes, due string) (*gtasks.Task, error) {
	var task *gtasks.Task
	err := retry(func() error {
		var err error
		task, err = s.svc.Tasks.Insert(s.taskListID, &gtasks.Task{Title:title,Notes:notes,Due:due}).Do()
		return err
	})
	return task, err
}

func (s *Service) Update(id, title, notes, status, due string) (*gtasks.Task, error) {
	var task *gtasks.Task
	err := retry(func() error {
		var err error
		task, err = s.svc.Tasks.Update(s.taskListID,id,&gtasks.Task{Title:title,Notes:notes,Status:status,Due:due}).Do()
		return err
	})
	return task, err
}

func (s *Service) Delete(id string) error {
	return retry(func() error {
		return s.svc.Tasks.Delete(s.taskListID,id).Do()
	})
}

func (s *Service) Clear() error {
	return retry(func() error {
		return s.svc.Tasks.Clear(s.taskListID).Do()
	})
}

func (s *Service) Get(id string) (*gtasks.Task, error) {
	var t *gtasks.Task
	err := retry(func() error {
		var err error
		t, err = s.svc.Tasks.Get(s.taskListID,id).Do()
		return err
	})
	return t, err
}
