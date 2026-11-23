package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
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

func (ir *ItemRepository) CreateItem(ctx context.Context, name string, price int, stock int) (Item, error) {
	query := "INSERT INTO items (name, price, stock) VALUES ($1, $2, $3) RETURNING id, name, price, created_at, stock"
	var item Item
	err := ir.pool.QueryRow(ctx, query, name, price, stock).Scan(&item.ID, &item.Name, &item.Price, &item.CreatedAt, &item.Stock)

	if err != nil {
		return Item{}, fmt.Errorf("error in CreateItem: %w", err)
	}

	return item, err
}

func (ir *ItemRepository) List(ctx context.Context) ([]Item, error) {
	query := "SELECT id, name, price, created_at FROM items ORDER BY id"
	itemSlice := make([]Item, 0)
	rows, err := ir.pool.Query(ctx, query)
	if err != nil {
		return []Item{}, fmt.Errorf("error in List: %w", err)
	}

	defer rows.Close()

	var item Item
	for rows.Next() {
		err := rows.Scan(&item.ID, &item.Name, &item.Price, &item.CreatedAt)
		if err != nil {
			return []Item{}, fmt.Errorf("error while scan: %w", err)
		}
		itemSlice = append(itemSlice, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error in rows: %w", err)
	}

	return itemSlice, nil
}

func (ir *ItemRepository) GetByID(ctx context.Context, id int) (Item, error) {
	query := "SELECT id, name, price, created_at FROM items WHERE id=$1"
	var item Item

	row := ir.pool.QueryRow(ctx, query, id)
	err := row.Scan(&item.ID, &item.Name, &item.Price, &item.CreatedAt)
	if err != nil {
		return Item{}, fmt.Errorf("error while scaning row in GetByID: %w", err)
	}
	return item, nil
}

func (ir *ItemRepository) Update(ctx context.Context, id int, name string, price int) (Item, error) {
	query := "UPDATE items SET name=$1, price=$2 WHERE id=$3 RETURNING id, name, price, created_at"
	var item Item
	row := ir.pool.QueryRow(ctx, query, name, price, id)
	err := row.Scan(&item.ID, &item.Name, &item.Price, &item.CreatedAt)
	if err != nil {
		return Item{}, fmt.Errorf("error while scanning row in Update: %w", err)
	}
	return item, nil
}

func (ir *ItemRepository) Delete(ctx context.Context, id int) error {
	query := "DELETE FROM items WHERE id=$1"

	tag, err := ir.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("error while deleting in Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
