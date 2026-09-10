package service

import (
	"context"
	"fmt"
	"log"
	"time"
	"todolist/internal/cache"
	"todolist/internal/event"
	"todolist/internal/kafka"
	"todolist/internal/model"
	"todolist/internal/repository"
)

type TodoService interface {
	Create(ctx context.Context, req model.CreateTodoRequest) (*model.Todo, error)
	GetByID(ctx context.Context, id int) (*model.Todo, error)
	GetAll(ctx context.Context, completed *bool) ([]model.Todo, error)
	Update(ctx context.Context, id int, req model.UpdateTodoRequest) (*model.Todo, error)
	Delete(ctx context.Context, id int) error
}

type todoService struct {
	repo     repository.TodoRepository
	cache    cache.Cache
	producer kafka.Producer
}

func NewTodoService(repo repository.TodoRepository, cache cache.Cache, producer kafka.Producer) TodoService {
	return &todoService{repo: repo, cache: cache, producer: producer}
}

func (s *todoService) Create(ctx context.Context, req model.CreateTodoRequest) (*model.Todo, error) {
	todo := &model.Todo{
		Title:       req.Title,
		Description: req.Description,
		Completed:   false,
	}
	if err := s.repo.Create(ctx, todo); err != nil {
		return nil, fmt.Errorf("failed to create todo: %w", err)
	}

	_ = s.cache.SetTodo(ctx, todo)
	_ = s.cache.InvalidateAll(ctx)

	s.publish(ctx, event.TodoCreated, todo)
	return todo, nil
}

func (s *todoService) GetByID(ctx context.Context, id int) (*model.Todo, error) {
	// Пробуем кеш
	if cached, err := s.cache.GetTodo(ctx, id); err == nil && cached != nil {
		return cached, nil
	}

	todo, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get todo: %w", err)
	}
	if todo == nil {
		return nil, fmt.Errorf("todo not found")
	}

	_ = s.cache.SetTodo(ctx, todo)
	return todo, nil
}

func (s *todoService) GetAll(ctx context.Context, completed *bool) ([]model.Todo, error) {
	// Кешируем только запрос без фильтра
	if completed == nil {
		if cached, err := s.cache.GetAllTodos(ctx); err == nil && cached != nil {
			return cached, nil
		}
	}

	todos, err := s.repo.GetAll(ctx, completed)
	if err != nil {
		return nil, fmt.Errorf("failed to get todos: %w", err)
	}

	if completed == nil {
		_ = s.cache.SetAllTodos(ctx, todos)
	}
	return todos, nil
}

func (s *todoService) Update(ctx context.Context, id int, req model.UpdateTodoRequest) (*model.Todo, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get todo: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("todo not found")
	}

	todo := &model.Todo{}
	if req.Title != nil {
		todo.Title = *req.Title
	}
	if req.Description != nil {
		todo.Description = *req.Description
	}
	if req.Completed != nil {
		todo.Completed = *req.Completed
	}

	if err := s.repo.Update(ctx, id, todo); err != nil {
		return nil, fmt.Errorf("failed to update todo: %w", err)
	}

	updated, _ := s.repo.GetByID(ctx, id)
	_ = s.cache.SetTodo(ctx, updated)
	_ = s.cache.InvalidateAll(ctx)

	s.publish(ctx, event.TodoUpdated, updated)
	return updated, nil
}

func (s *todoService) Delete(ctx context.Context, id int) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get todo: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("todo not found")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	_ = s.cache.DeleteTodo(ctx, id)
	_ = s.cache.InvalidateAll(ctx)

	s.publish(ctx, event.TodoDeleted, &model.Todo{ID: id})
	return nil
}

func (s *todoService) publish(ctx context.Context, evtType event.TodoEventType, todo *model.Todo) {
	evt := event.TodoEvent{
		Type:      evtType,
		TodoID:    todo.ID,
		Todo:      todo,
		Timestamp: time.Now().Unix(),
	}
	if err := s.producer.PublishEvent(ctx, evt); err != nil {
		log.Printf("failed to publish event: %v", err)
	}
}
