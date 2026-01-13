package examples

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/n-r-w/pgh/v2"
	"github.com/n-r-w/pgh/v2/pq"
	sq "github.com/n-r-w/squirrel"
)

// ExamplePQSelectOne demonstrates using pq.SelectOne to fetch a single user.
func ExamplePQSelectOne(ctx context.Context, db pq.IQuerier, userID int) (*User, error) {
	query := pgh.Builder().
		Select("*").
		From("users").
		Where("id = ?", userID)

	var user User
	if err := pq.SelectOne(ctx, db, query, &user); err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	return &user, nil
}

// ExamplePQSelect demonstrates using pq.Select to fetch multiple users.
func ExamplePQSelect(ctx context.Context, db pq.IQuerier, userID int) ([]User, error) {
	query := pgh.Builder().
		Select("*").
		From("users").
		Where("id = ?", userID)

	var users []User
	if err := pq.Select(ctx, db, query, &users); err != nil {
		return nil, fmt.Errorf("get users: %w", err)
	}

	return users, nil
}

// ExamplePQExec demonstrates using pq.Exec to update a user.
func ExamplePQExec(ctx context.Context, db pq.IQuerier, user *User) error {
	query := pgh.Builder().
		Update("users").
		SetMap(map[string]any{
			"name":  user.Name,
			"email": user.Email,
		}).
		Where(sq.Eq{"id": user.ID})

	_, err := pq.Exec(ctx, db, query)
	return err
}

// ExamplePQSelectFunc demonstrates using pq.SelectFunc to process rows one at a time.
func ExamplePQSelectFunc(ctx context.Context, db pq.IQuerier) error {
	query := pgh.Builder().
		Select("id, name, email, status, bio, last_login").
		From("users").
		Where("status = ?", "active")

	return pq.SelectFunc(ctx, db, query, func(rows *sql.Rows) error {
		var user User
		err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.Status,
			&user.Bio,
			&user.LastLogin,
		)
		if err != nil {
			return fmt.Errorf("scan user: %w", err)
		}
		// Process each user here
		//nolint:forbidigo // example code
		fmt.Printf("Processing user: %s\n", user.Name)
		return nil
	})
}
