// Package filter ...
// Deprecated: use github.com/n-r-w/squirrel
package filter

import (
	"fmt"
	"slices"

	sq "github.com/n-r-w/squirrel"
)

type option func(f *filter)

// WithPKField specifies the name for PK. Used in cases where PK name differs from default "id".
// Can specify a composite key in the format: (field1, field2)
func WithPKField(pkfield string) option {
	return func(f *filter) {
		f.pkField = pkfield
	}
}

// WithIn adds an IN condition for sqlName with values
func WithIn[T any](sqlName string, values []T) option {
	return func(f *filter) {
		f.inConds = append(f.inConds, *NewInCond(sqlName, values))
	}
}

// WithInConds adds IN conditions. Useful in cases where you need to customize the created InCond condition
func WithInConds(conds ...inCond) option {
	return func(f *filter) {
		f.inConds = append(f.inConds, conds...)
	}
}

// WithOrders adds sorting conditions (orders).
// Aliases for mapping orderID from OrderCond and sqlName are passed in aliases parameter.
// Duplicate sort conditions are ignored
func WithOrders(aliases map[int]string, orders ...OrderCond) option {
	return func(f *filter) {
		f.orderAliases = aliases
		for _, oc := range orders {
			if slices.ContainsFunc(f.orders, func(o OrderCond) bool {
				return o.orderID == oc.orderID
			}) {
				continue
			}

			f.orders = append(f.orders, oc)
		}
	}
}

// WithoutOrder forcibly disables sorting
func WithoutOrder() option {
	return func(f *filter) {
		f.withoutOrder = true
	}
}

// WithSearch adds search condition for all provided fields
func WithSearch(search string, fields ...string) option {
	return func(f *filter) {
		f.search = search
		f.searchFields = fields
	}
}

// WithPaginator adds pagination based on Paginator.
// If a service paginator is provided, pagination will be performed by the last PK value with condition pkField > lastID
// If a frontend paginator is provided, pagination will be performed by limit and offset.
// Panics when attempting to add paginator again
func WithPaginator(paginator Paginator) option {
	return func(f *filter) {
		if !f.paginator.isEmpty() {
			panic("attempt to re-add paginator")
		}
		f.paginator = paginator
	}
}

type filter struct {
	inConds []inCond

	orders       []OrderCond
	orderAliases map[int]string
	withoutOrder bool

	search       string
	searchFields []string
	paginator    Paginator

	// PK field. Defaults to "id". When no explicit sorting is provided, sorts by this field.
	// Can use a composite key, for example, (product_id, repository_id).
	pkField string
}

func newFilter(opts ...option) (*filter, error) {
	f := &filter{
		pkField: "id",
	}

	for _, o := range opts {
		o(f)
	}

	return f, validateFilter(f)
}

func validateFilter(f *filter) error {
	for _, oc := range f.orders {
		_, ok := f.orderAliases[oc.orderID]
		if !ok {
			return fmt.Errorf("get sql alias for orderID=%d: %w", oc.orderID, ErrOrderAliasNotFound)
		}
	}

	return nil
}

// NewSelectBuilder returns squirrel SelectBuilder built from provided opts
// Deprecated: use github.com/n-r-w/squirrel
func NewSelectBuilder(opts ...option) (sq.SelectBuilder, error) {
	f, err := newFilter(opts...)
	if err != nil {
		return sq.Select(), err
	}

	return newSquirrelBuilder(f).build()
}
