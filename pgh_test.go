package pgh

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_TruncSQL(t *testing.T) {
	t.Parallel()

	var sqlOK, sqlTrunc string
	for range sqlTruncLen {
		sqlOK += "1"
		sqlTrunc += "1"
	}
	sqlTrunc += "1"

	require.Len(t, TruncSQL(sqlTrunc), sqlTruncLen+3)
	require.Equal(t, sqlOK, TruncSQL(sqlOK))
}
