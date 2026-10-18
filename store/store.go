package store

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrTaskNotFound = errors.New("task not found")
	ErrInvalidTask  = errors.New("invalid task")
)

type TaskStatus string

const (
	StatusPending    TaskStatus = "pending"
	StatusInProgress TaskStatus = "in_progress"
	StatusCompleted  TaskStatus = "completed"
)

func (s TaskStatus) Valid() bool {
	switch s {
	case StatusPending, StatusInProgress, StatusCompleted:
		return true
	default:
		return false
	}
}

type Task struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
}

type CreateTaskInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdateTaskInput struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
}

type TaskStore struct {
	mu     sync.RWMutex
	nextID int64
	tasks  map[int64]Task
}

func NewTaskStore() *TaskStore {
	return &TaskStore{
		nextID: 1,
		tasks:  make(map[int64]Task),
	}
}

func (s *TaskStore) Create(input CreateTaskInput) (Task, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" || len([]rune(title)) > 120 || len([]rune(input.Description)) > 2000 {
		return Task{}, ErrInvalidTask
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	task := Task{
		ID:          s.nextID,
		Title:       title,
		Description: input.Description,
		Status:      StatusPending,
		CreatedAt:   time.Now().UTC(),
	}
	s.nextID++
	s.tasks[task.ID] = task
	return task, nil
}

func (s *TaskStore) Get(id int64) (Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[id]
	if !ok {
		return Task{}, ErrTaskNotFound
	}
	return task, nil
}

func (s *TaskStore) List() []Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	return tasks
}

func (s *TaskStore) Update(id int64, input UpdateTaskInput) (Task, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" || len([]rune(title)) > 120 || len([]rune(input.Description)) > 2000 || !input.Status.Valid() {
		return Task{}, ErrInvalidTask
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[id]
	if !ok {
		return Task{}, ErrTaskNotFound
	}

	task.Title = title
	task.Description = input.Description
	task.Status = input.Status
	s.tasks[id] = task
	return task, nil
}

func (s *TaskStore) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; !ok {
		return ErrTaskNotFound
	}
	delete(s.tasks, id)
	return nil
}
