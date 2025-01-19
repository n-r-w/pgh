package filter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type SquirrelBuilderSuite struct {
	suite.Suite
}

// normalizeSQL converts input into a single string without duplicate spaces
func (s *SquirrelBuilderSuite) normalizeSQL(sql string) string {
	fields := strings.Fields(sql)
	return strings.Join(fields, " ")
}

func (s *SquirrelBuilderSuite) TestBuild_filter() {
	ids := []int{0, 1, 2, 3, 4}
	names := []string{"first", "second", ""}
	searchValue := "search"

	sqBuilder, err := NewSelectBuilder(
		WithIn("id", ids),
		WithInConds(NewInCond("name", names).WithZeroValue(false)), // all empty values will be ignored
		WithIn("empty", []string{}),
		WithSearch(searchValue, "id", "name"),
	)
	s.Require().NoError(err)

	sql, vals, err := sqBuilder.Columns("*").From("table").ToSql()
	s.Require().NoError(err)

	expectesValues := []any{
		0, 1, 2, 3, 4,
		"first", "second",
		"%" + searchValue + "%", "%" + searchValue + "%",
	}

	s.Require().Equal(expectesValues, vals)

	expectedSQL := s.normalizeSQL(`
		SELECT * 
		FROM table 
		WHERE id IN ($1,$2,$3,$4,$5) AND name IN ($6,$7) AND (id::text LIKE $8 OR name::text LIKE $9)
		ORDER BY id ASC
	`)
	s.Require().Equal(expectedSQL, sql)
}

func (s *SquirrelBuilderSuite) TestBuild_order() {
	s.Run("With explicit orders", func() {
		const (
			UserID = iota
			UserName
		)

		orders := []OrderCond{
			*NewOrder(UserID, ASC),
			*NewOrder(UserName, DESC),
			*NewOrder(UserID, ASC), // Duplicate should be ignored
		}

		sqBuilder, err := NewSelectBuilder(
			WithOrders(map[int]string{
				UserID:   "id",
				UserName: "name",
			}, orders...),
		)
		s.Require().NoError(err)

		sql, _, err := sqBuilder.Columns("*").From("table").ToSql()
		s.Require().NoError(err)

		expectedSQL := s.normalizeSQL(`
			SELECT * 
			FROM table 
			ORDER BY id ASC, name DESC
		`)
		s.Require().Equal(expectedSQL, sql)
	})

	s.Run("With order by custom composite PK", func() {
		sqBuilder, err := NewSelectBuilder(
			WithPKField("(field1, field2)"),
		)
		s.Require().NoError(err)

		sql, _, err := sqBuilder.Columns("*").From("table").ToSql()
		s.Require().NoError(err)

		expectedSQL := s.normalizeSQL(`
			SELECT * 
			FROM table 
			ORDER BY (field1, field2) ASC
		`)
		s.Require().Equal(expectedSQL, sql)
	})
}

func (s *SquirrelBuilderSuite) TestBuild_paginator() {
	s.Run("Front paginator", func() {
		paginator := NewFrontPaginator(10, 2)
		sqBuilder, err := NewSelectBuilder(
			WithPaginator(*paginator),
		)
		s.Require().NoError(err)

		sql, _, err := sqBuilder.Columns("*").From("table").ToSql()
		s.Require().NoError(err)

		expectedSQL := s.normalizeSQL(`
			SELECT * 
			FROM table
			ORDER BY id ASC
			LIMIT 10
			OFFSET 10
		`)
		s.Require().Equal(expectedSQL, sql)
	})

	s.Run("Service paginator", func() {
		paginator := NewServicePaginator(10, 10)
		sqBuilder, err := NewSelectBuilder(
			WithPaginator(*paginator),
		)
		s.Require().NoError(err)

		sql, vals, err := sqBuilder.Columns("*").From("table").ToSql()
		s.Require().NoError(err)

		s.Require().Equal([]any{uint64(10)}, vals)

		expectedSQL := s.normalizeSQL(`
			SELECT * 
			FROM table
			WHERE id > $1
			ORDER BY id ASC
			LIMIT 10
		`)
		s.Require().Equal(expectedSQL, sql)
	})

	s.Run("Double adding paginator", func() {
		s.Panics(func() {
			_, err := NewSelectBuilder(
				WithPaginator(*NewServicePaginator(0, 0)),
				WithPaginator(*NewFrontPaginator(0, 0)),
			)
			s.Require().NoError(err)
		})
	})
}

func TestSquirrelBuilderSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SquirrelBuilderSuite))
}
