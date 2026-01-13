package examples

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/n-r-w/pgh/v2"
	"github.com/n-r-w/pgh/v2/px"
	sq "github.com/n-r-w/squirrel"
)

type User struct {
	ID        int    `db:"id"`
	Name      string `db:"name"`
	Email     string `db:"email"`
	Status    string `db:"status"`
	Bio       string `db:"bio"`
	LastLogin string `db:"last_login"`
}

func ExampleSelectOne(ctx context.Context, db px.IQuerier, userID int) (*User, error) {
	query := pgh.Builder().
		Select("*").
		From("users").
		Where("id = ?", userID)

	var user User
	if err := px.SelectOne(ctx, db, query, &user); err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	return &user, nil
}

func ExampleSelectMany(ctx context.Context, db px.IQuerier, userID int) ([]User, error) {
	query := pgh.Builder().
		Select("*").
		From("users")

	var users []User
	if err := px.Select(ctx, db, query, &users); err != nil {
		return nil, fmt.Errorf("get users: %w", err)
	}

	return users, nil
}

func ExampleExec(ctx context.Context, tx px.IQuerier, user *User) error {
	query := pgh.Builder().
		Update("users").
		SetMap(map[string]any{
			"name":  user.Name,
			"email": user.Email,
		}).
		Where(sq.Eq{"id": user.ID})

	_, err := px.Exec(ctx, tx, query)
	return err
}

// ExampleBatchOperations demonstrates batch operations
func ExampleBatchOperations(ctx context.Context, db px.IBatcher, users []User) error {
	queries := make([]sq.Sqlizer, 0, len(users))

	for _, user := range users {
		query := pgh.Builder().
			Insert("users").
			SetMap(map[string]any{
				"name":  user.Name,
				"email": user.Email,
			})
		queries = append(queries, query)
	}

	_, err := px.ExecBatch(ctx, queries, db)
	return err
}

// ExampleErrorHandling demonstrates error handling
func ExampleErrorHandling(err error) {
	if err != nil {
		switch {
		case px.IsNoRows(err):
			// Handle no rows found
			// Handles both pgx.ErrNoRows and PostgreSQL 'no_data_found' error code
			fmt.Println("No rows found")
		case px.IsUniqueViolation(err):
			// Handle unique constraint violation
			// Maps to PostgreSQL error code '23505'
			fmt.Println("Unique constraint violation")
		case px.IsForeignKeyViolation(err):
			// Handle foreign key violation
			// Maps to PostgreSQL error code '23503'
			fmt.Println("Foreign key violation")
		default:
			fmt.Printf("Unknown error: %v\n", err)
		}
	}
}

// ExampleSelectFunc demonstrates using SelectFunc to process rows one at a time
func ExampleSelectFunc(ctx context.Context, db px.IQuerier) error {
	query := pgh.Builder().
		Select("id, name, email, status, bio, last_login").
		From("users").
		Where("status = ?", "active")

	return px.SelectFunc(ctx, db, query, func(row pgx.Row) error {
		var user User
		err := row.Scan(
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
		fmt.Printf("Processing user: %s\n", user.Name)
		return nil
	})
}

// ExampleExecSplit demonstrates splitting large updates into smaller batches
func ExampleExecSplit(ctx context.Context, db px.IBatcher, users []User) (int64, error) {
	queries := make([]sq.Sqlizer, 0, len(users))

	for _, user := range users {
		query := pgh.Builder().
			Update("users").
			SetMap(map[string]any{
				"status": "inactive",
			}).
			Where(sq.Eq{"id": user.ID})
		queries = append(queries, query)
	}

	// Split into batches of 100 updates
	return px.ExecSplit(ctx, db, queries, 100)
}

// ExampleInsertSplit demonstrates inserting large datasets in batches
func ExampleInsertSplit(ctx context.Context, db px.IBatcher, users []User) (int64, error) {
	baseQuery := pgh.Builder().
		Insert("users").
		Columns("name", "email", "status")

	values := make([]pgh.Args, 0, len(users))
	for _, user := range users {
		values = append(values, pgh.Args{user.Name, user.Email, user.Status})
	}

	// Insert in batches of 100 rows
	return px.InsertSplit(ctx, db, baseQuery, values, 100)
}

// ExampleInsertSplitQuery demonstrates inserting data in batches and retrieving inserted rows
func ExampleInsertSplitQuery(ctx context.Context, db px.IBatcher, users []User) ([]User, error) {
	baseQuery := pgh.Builder().
		Insert("users").
		Columns("name", "email", "status").
		Suffix("RETURNING *")

	values := make([]pgh.Args, 0, len(users))
	for _, user := range users {
		values = append(values, pgh.Args{user.Name, user.Email, user.Status})
	}

	var inserted []User
	err := px.InsertSplitQuery(ctx, db, baseQuery, values, 100, &inserted)
	if err != nil {
		return nil, fmt.Errorf("insert users: %w", err)
	}

	return inserted, nil
}

// ExampleSelectBatch demonstrates executing multiple select queries in a batch
func ExampleSelectBatch(ctx context.Context, db px.IBatcher, userIDs []int) ([]User, error) {
	queries := make([]sq.Sqlizer, 0, len(userIDs))

	for _, id := range userIDs {
		query := pgh.Builder().
			Select("*").
			From("users").
			Where(sq.Eq{"id": id})
		queries = append(queries, query)
	}

	var users []User
	if err := px.SelectBatch(ctx, queries, db, &users); err != nil {
		return nil, fmt.Errorf("select users batch: %w", err)
	}

	return users, nil
}
