package filter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateFilter(t *testing.T) {
	t.Run("Invalid filter because of missing order alias", func(t *testing.T) {
		const (
			userID = iota
			userName
		)

		orders := []OrderCond{
			*NewOrder(userID, DESC),
			*NewOrder(userName, DESC),
		}
		_, err := newFilter(
			WithOrders(map[int]string{userID: "id"}, orders...),
		)
		require.Error(t, err)
	})
}
