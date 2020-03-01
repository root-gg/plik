package middleware

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/root-gg/utils"
	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

func TestPaginateDefault(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/something", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	Paginate(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestOK(t, rr)

	require.NotNil(t, ctx.GetPagingQuery(), "missing paging query")
	require.Equal(t, 20, *ctx.GetPagingQuery().Limit, "invalid limit")
	require.Nil(t, ctx.GetPagingQuery().Order, "invalid order")
	require.Nil(t, ctx.GetPagingQuery().After, "invalid after")
	require.Nil(t, ctx.GetPagingQuery().Before, "invalid before")
}

func TestPaginate(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/something?limit=1&order=asc&after=after", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	Paginate(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestOK(t, rr)

	require.NotNil(t, ctx.GetPagingQuery(), "missing paging query")
	require.Equal(t, 1, *ctx.GetPagingQuery().Limit, "invalid limit")
	require.Equal(t, "asc", *ctx.GetPagingQuery().Order, "invalid order")
	require.Equal(t, "after", *ctx.GetPagingQuery().After, "invalid after")
	require.Nil(t, ctx.GetPagingQuery().Before, "invalid before")
}

func TestPaginateBefore(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/something?limit=1&order=asc&before=before", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	Paginate(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestOK(t, rr)

	require.NotNil(t, ctx.GetPagingQuery(), "missing paging query")
	require.Equal(t, 1, *ctx.GetPagingQuery().Limit, "invalid limit")
	require.Equal(t, "asc", *ctx.GetPagingQuery().Order, "invalid order")
	require.Equal(t, "before", *ctx.GetPagingQuery().Before, "invalid before")
	require.Nil(t, ctx.GetPagingQuery().After, "invalid after")
}

func TestPaginateHeader(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/something", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	pagingQuery := common.NewPagingQuery().WithLimit(1).WithOrder("asc").WithAfterCursor("after")
	req.Header.Set("X-Plik-Paging", utils.Sdump(pagingQuery))

	rr := ctx.NewRecorder(req)
	Paginate(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestOK(t, rr)

	require.NotNil(t, ctx.GetPagingQuery(), "missing paging query")
	require.Equal(t, 1, *ctx.GetPagingQuery().Limit, "invalid limit")
	require.Equal(t, "asc", *ctx.GetPagingQuery().Order, "invalid order")
	require.Equal(t, "after", *ctx.GetPagingQuery().After, "invalid after")
	require.Nil(t, ctx.GetPagingQuery().Before, "invalid before")
}

func TestPaginateInvalidHeader(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/something", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("X-Plik-Paging", "blah blah blah")

	rr := ctx.NewRecorder(req)
	Paginate(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestBadRequest(t, rr, "invalid paging header")
}

func TestPaginateInvalidLimit(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/something?limit=-1", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	Paginate(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestBadRequest(t, rr, "invalid limit")
}

func TestPaginateInvalidLimitParsing(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/something?limit=limit", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	Paginate(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestBadRequest(t, rr, "invalid limit")
}

func TestPaginateInvalidOrder(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/something?order=blah", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	Paginate(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestBadRequest(t, rr, "invalid order")
}

func TestPaginateBothCursors(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/something?before=before&after=after", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	Paginate(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestBadRequest(t, rr, "both before and after cursors set")
}
