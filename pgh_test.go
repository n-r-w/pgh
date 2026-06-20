package pgh

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_TruncSQL(t *testing.T) {
	t.Parallel()

	sqlOK := strings.Repeat("1", sqlTruncLen)
	sqlTrunc := strings.Repeat("1", sqlTruncLen+1)

	require.Len(t, TruncSQL(sqlTrunc), sqlTruncLen+3)
	require.Equal(t, sqlOK, TruncSQL(sqlOK))
}
