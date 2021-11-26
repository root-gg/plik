package common

import (
	"testing"

	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"
	"github.com/stretchr/testify/require"
)

func TestNewPagingQuery(t *testing.T) {
	pagingQuery := NewPagingQuery().
		WithLimit(10).
		WithOrder("asc").
		WithBeforeCursor("before").
		WithAfterCursor("after")

	require.NotNil(t, pagingQuery)
	require.Equal(t, 10, *pagingQuery.Limit)
	require.Equal(t, "asc", *pagingQuery.Order)
	require.Equal(t, "before", *pagingQuery.Before)
	require.Equal(t, "after", *pagingQuery.After)
}

func TestPagingQuery_Paginator(t *testing.T) {
	pagingQuery := NewPagingQuery().
		WithLimit(10).
		WithOrder("asc").
		WithBeforeCursor("before").
		WithAfterCursor("after")

	paginator := pagingQuery.Paginator()
	require.NotNil(t, paginator)
}

func TestNewPagingResponse(t *testing.T) {
	results := append([]interface{}{}, 1, 2, 3)
	before := "before"
	after := "after"
	pagingResponse := NewPagingResponse(results, &paginator.Cursor{Before: &before, After: &after})
	require.NotNil(t, pagingResponse)
	require.Equal(t, before, *pagingResponse.Before)
	require.Equal(t, after, *pagingResponse.After)
	require.Len(t, pagingResponse.Results, 3)
}
