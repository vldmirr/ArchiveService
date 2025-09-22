package repository

import (
    "archive-service/models"
    "sync"
)

type Storage struct {
    tasks map[string]*models.Task
    mutex sync.RWMutex
}

func NewStorage() *Storage {
    return &Storage{
        tasks: make(map[string]*models.Task),
    }
}

func (s *Storage) CreateTask(task *models.Task) {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    s.tasks[task.ID] = task
}

func (s *Storage) GetTask(taskID string) (*models.Task, bool) {
    s.mutex.RLock()
    defer s.mutex.RUnlock()
    task, exists := s.tasks[taskID]
    return task, exists
}

func (s *Storage) UpdateTask(task *models.Task) {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    s.tasks[task.ID] = task
}

func (s *Storage) GetAllTasks() []*models.Task {
    s.mutex.RLock()
    defer s.mutex.RUnlock()
    
    tasks := make([]*models.Task, 0, len(s.tasks))
    for _, task := range s.tasks {
        tasks = append(tasks, task)
    }
    return tasks
}