package pgh

import (
	sq "github.com/n-r-w/squirrel"
)

// Args is a slice of values for binding.
// Used to explicitly separate query parameters from other arguments.
type Args []any

// Builder creates a new instance of squirrel.StatementBuilderType for building queries
func Builder() sq.StatementBuilderType {
	return sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
}

const sqlTruncLen = 100

// TruncSQL truncates sql to sqlTruncLen characters.
func TruncSQL(sql string) string {
	if len(sql) > sqlTruncLen {
		return sql[0:sqlTruncLen] + "..."
	}

	return sql
}
