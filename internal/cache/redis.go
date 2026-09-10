package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"todolist/internal/config"
	"todolist/internal/model"

	"github.com/redis/go-redis/v9"
)

type Cache interface {
	GetTodo(ctx context.Context, id int) (*model.Todo, error)
	SetTodo(ctx context.Context, todo *model.Todo) error
	DeleteTodo(ctx context.Context, id int) error
	GetAllTodos(ctx context.Context) ([]model.Todo, error)
	SetAllTodos(ctx context.Context, todos []model.Todo) error
	InvalidateAll(ctx context.Context) error
}

type redisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func New(cfg *config.Config) Cache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	return &redisCache{client: rdb, ttl: 5 * time.Minute}
}

func (c *redisCache) GetTodo(ctx context.Context, id int) (*model.Todo, error) {
	val, err := c.client.Get(ctx, fmt.Sprintf("todo:%d", id)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var todo model.Todo
	if err := json.Unmarshal([]byte(val), &todo); err != nil {
		return nil, err
	}
	return &todo, nil
}

func (c *redisCache) SetTodo(ctx context.Context, todo *model.Todo) error {
	data, err := json.Marshal(todo)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, fmt.Sprintf("todo:%d", todo.ID), data, c.ttl).Err()
}

func (c *redisCache) DeleteTodo(ctx context.Context, id int) error {
	return c.client.Del(ctx, fmt.Sprintf("todo:%d", id)).Err()
}

func (c *redisCache) GetAllTodos(ctx context.Context) ([]model.Todo, error) {
	val, err := c.client.Get(ctx, "todos:all").Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var todos []model.Todo
	if err := json.Unmarshal([]byte(val), &todos); err != nil {
		return nil, err
	}
	return todos, nil
}

func (c *redisCache) SetAllTodos(ctx context.Context, todos []model.Todo) error {
	data, err := json.Marshal(todos)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, "todos:all", data, c.ttl).Err()
}

func (c *redisCache) InvalidateAll(ctx context.Context) error {
	return c.client.Del(ctx, "todos:all").Err()
}
