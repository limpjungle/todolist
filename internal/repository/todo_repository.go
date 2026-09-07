package repository

import (
	"context"
	"todolist/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TodoRepository interface {
	Create(ctx context.Context, todo *model.Todo) error
	GetByID(ctx context.Context, id int) (*model.Todo, error)
	GetAll(ctx context.Context, completed *bool) ([]model.Todo, error)
	Update(ctx context.Context, id int, todo *model.Todo) error
	Delete(ctx context.Context, id int) error
}

type todoRepository struct {
	db *pgxpool.Pool
}

func NewTodoRepository(db *pgxpool.Pool) TodoRepository {
	return &todoRepository{db: db}
}

func (r *todoRepository) Create(ctx context.Context, todo *model.Todo) error {
	query := `
		INSERT INTO todos (title, description, completed)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query, todo.Title, todo.Description, todo.Completed).
		Scan(&todo.ID, &todo.CreatedAt, &todo.UpdatedAt)
}

func (r *todoRepository) GetByID(ctx context.Context, id int) (*model.Todo, error) {
	query := `
		SELECT id, title, description, completed, created_at, updated_at
		FROM todos WHERE id = $1
	`
	var todo model.Todo
	err := r.db.QueryRow(ctx, query, id).Scan(
		&todo.ID, &todo.Title, &todo.Description,
		&todo.Completed, &todo.CreatedAt, &todo.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &todo, nil
}

func (r *todoRepository) GetAll(ctx context.Context, completed *bool) ([]model.Todo, error) {
	query := `
		SELECT id, title, description, completed, created_at, updated_at
		FROM todos
	`
	var args []interface{}
	if completed != nil {
		query += " WHERE completed = $1"
		args = append(args, *completed)
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []model.Todo
	for rows.Next() {
		var todo model.Todo
		err := rows.Scan(
			&todo.ID, &todo.Title, &todo.Description,
			&todo.Completed, &todo.CreatedAt, &todo.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}
	return todos, rows.Err()
}

func (r *todoRepository) Update(ctx context.Context, id int, todo *model.Todo) error {
	query := `
		UPDATE todos 
		SET title = COALESCE(NULLIF($1, ''), title),
		    description = COALESCE($2, description),
		    completed = COALESCE($3, completed),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $4
		RETURNING title, description, completed, updated_at
	`
	return r.db.QueryRow(ctx, query, todo.Title, todo.Description, todo.Completed, id).
		Scan(&todo.Title, &todo.Description, &todo.Completed, &todo.UpdatedAt)
}

func (r *todoRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM todos WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
