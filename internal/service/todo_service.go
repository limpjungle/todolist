package service

import (
	"context"
	"fmt"
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
	repo repository.TodoRepository
}

func NewTodoService(repo repository.TodoRepository) TodoService {
	return &todoService{repo: repo}
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
	return todo, nil
}

func (s *todoService) GetByID(ctx context.Context, id int) (*model.Todo, error) {
	todo, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get todo: %w", err)
	}
	if todo == nil {
		return nil, fmt.Errorf("todo not found")
	}
	return todo, nil
}

func (s *todoService) GetAll(ctx context.Context, completed *bool) ([]model.Todo, error) {
	todos, err := s.repo.GetAll(ctx, completed)
	if err != nil {
		return nil, fmt.Errorf("failed to get todos: %w", err)
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

	// Получаем обновленную запись
	return s.repo.GetByID(ctx, id)
}

func (s *todoService) Delete(ctx context.Context, id int) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get todo: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("todo not found")
	}
	return s.repo.Delete(ctx, id)
}
