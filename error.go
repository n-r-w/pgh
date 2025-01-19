package pgh

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Helpers for working with Postgres errors.
// The pgerrcode package contains Postgres error codes and many useful functions like IsIntegrityConstraintViolation.
// If something is missing there, it is added to this file.
// https://www.postgresql.org/docs/16/errcodes-appendix.html

// IsNoRows checks if the error is a "no rows" error.
func IsNoRows(err error) bool {
	if errors.Is(err, pgx.ErrNoRows) {
		return true
	}

	if pgErr, ok := toPgError(err); ok {
		if pgErr.Code == pgerrcode.NoDataFound {
			return true
		}
	}
	return false
}

// IsUniqueViolation checks if the error is a unique constraint violation.
func IsUniqueViolation(err error) bool {
	if pgErr, ok := toPgError(err); ok {
		if pgErr.Code == pgerrcode.UniqueViolation {
			return true
		}
	}
	return false
}

// IsForeignKeyViolation checks if the error is a foreign key constraint violation.
func IsForeignKeyViolation(err error) bool {
	if pgErr, ok := toPgError(err); ok {
		if pgErr.Code == pgerrcode.ForeignKeyViolation {
			return true
		}
	}
	return false
}

func toPgError(err error) (*pgconn.PgError, bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr, true
	}
	return nil, false
}
