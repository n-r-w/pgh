package filter

// inCond represents an IN condition in SQL
type inCond struct {
	sqlName      string
	values       []any
	useZeroValue bool
}

// NewInCond returns an object for creating an IN condition in SQL
func NewInCond[T any](sqlName string, values []T) *inCond {
	anyValues := []any{}
	for _, v := range values {
		anyValues = append(anyValues, v)
	}

	return &inCond{
		values:       anyValues,
		useZeroValue: true,
		sqlName:      sqlName,
	}
}

// WithZeroValue sets a parameter for using zero values on the source object.
// If use=true, then all values of the source object will be
// used in the query. If use=false, then all empty values in values (nil, "", 0)
// will be ignored when building the query
func (i inCond) WithZeroValue(use bool) inCond {
	i.useZeroValue = use
	return i
}
