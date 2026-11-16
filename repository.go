package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ItemRepository struct {
	pool *pgxpool.Pool
}

func NewItemRepository(pool *pgxpool.Pool) *ItemRepository {
	return &ItemRepository{
		pool: pool,
	}
}

func (ir *ItemRepository) CreateItem(ctx context.Context, name string, price int) (Item, error) {
	query := "INSERT INTO items (name, price) VALUES ($1, $2) RETURNING id, name, price, created_at"
	var (
		r_id         int
		r_name       string
		r_price      int
		r_created_at time.Time
	)
	err := ir.pool.QueryRow(ctx, query, name, price).Scan(&r_id, &r_name, &r_price, &r_created_at)

	if err != nil {
		return Item{}, fmt.Errorf("error in CreateItem: %w", err)
	}

	return Item{
		ID:        r_id,
		Name:      r_name,
		Price:     r_price,
		CreatedAt: r_created_at,
	}, err
}

func (ir *ItemRepository) List(ctx context.Context) ([]Item, error) {
	query := "SELECT id, name, price, created_at FROM items ORDER BY id"
	itemSlice := make([]Item, 0)
	rows, err := ir.pool.Query(ctx, query)
	if err != nil {
		return []Item{}, fmt.Errorf("error in List: %w", err)
	}

	defer rows.Close()

	var (
		r_id         int
		r_name       string
		r_price      int
		r_created_at time.Time
	)
	for rows.Next() {
		err := rows.Scan(&r_id, &r_name, &r_price, &r_created_at)
		if err != nil {
			return []Item{}, fmt.Errorf("error while scan: %w", err)
		}
		itemSlice = append(itemSlice, Item{
			ID:        r_id,
			Name:      r_name,
			Price:     r_price,
			CreatedAt: r_created_at,
		})
	}
	if err := rows.Err(); err != nil {
		return []Item{}, fmt.Errorf("error in rows: %w", err)
	}

	return itemSlice, nil
}
