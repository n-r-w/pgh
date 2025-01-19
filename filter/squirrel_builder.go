package filter

import (
	"fmt"
	"reflect"

	"github.com/n-r-w/pgh"
	sq "github.com/n-r-w/squirrel"
)

type squirrelBuilder struct {
	f *filter
}

func newSquirrelBuilder(f *filter) *squirrelBuilder {
	return &squirrelBuilder{
		f: f,
	}
}

func (s *squirrelBuilder) build() (sq.SelectBuilder, error) {
	builder := pgh.Builder().Select()

	for _, ord := range s.f.orders {
		sql, ok := s.f.orderAliases[ord.orderID]
		if !ok {
			continue
		}
		switch ord.Type {
		case ASC:
			builder = builder.OrderBy(sql + " " + ASC.String())
		case DESC:
			builder = builder.OrderBy(sql + " " + DESC.String())
		default:
			return builder, fmt.Errorf("order type '%d': %w", ord.Type, ErrUnknownOrderType)
		}
	}

	if len(s.f.orders) == 0 && !s.f.withoutOrder {
		builder = builder.OrderBy(s.f.pkField + " " + ASC.String())
	}

	for _, cond := range s.f.inConds {
		if len(cond.values) == 0 {
			continue
		}

		values := []any{}
		if cond.useZeroValue {
			values = cond.values
		} else {
			for _, v := range cond.values {
				if !reflect.ValueOf(v).IsZero() {
					values = append(values, v)
				}
			}
		}

		if len(values) != 0 {
			builder = builder.Where(sq.Eq{cond.sqlName: values})
		}
	}

	if s.f.search != "" {
		search := sq.Or{}
		for _, v := range s.f.searchFields {
			search = append(search, sq.Like{v + "::text": "%" + s.f.search + "%"})
		}

		if len(search) != 0 {
			builder = builder.Where(search)
		}
	}

	if s.f.paginator.limit > 0 {
		builder = builder.Limit(uint64(s.f.paginator.limit))
	}

	if s.f.paginator.IsService() {
		builder = builder.Where(sq.Gt{s.f.pkField: s.f.paginator.lastID})
	} else if s.f.paginator.offset > 0 {
		builder = builder.Offset(s.f.paginator.offset)
	}

	return builder, nil
}
