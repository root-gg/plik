package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

// Paginate parse pagination requests
func Paginate(ctx *context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {

		pagingQuery := common.NewPagingQuery().WithLimit(20)

		header := req.Header.Get("X-Plik-Paging")
		if header != "" {
			err := json.Unmarshal([]byte(header), &pagingQuery)
			if err != nil {
				ctx.InvalidParameter("paging header")
				return
			}
		} else {
			limitStr := req.URL.Query().Get("limit")
			if limitStr != "" {
				limit, err := strconv.Atoi(limitStr)
				if err != nil {
					ctx.InvalidParameter("limit : %s", err)
				}
				pagingQuery.WithLimit(limit)
			}

			order := req.URL.Query().Get("order")
			if order != "" {
				pagingQuery.WithOrder(order)
			}

			before := req.URL.Query().Get("before")
			if before != "" {
				pagingQuery.WithBeforeCursor(before)
			}

			after := req.URL.Query().Get("after")
			if after != "" {
				pagingQuery.WithAfterCursor(after)
			}
		}

		if pagingQuery.Limit != nil && *pagingQuery.Limit <= 0 {
			ctx.InvalidParameter("limit")
		}

		if pagingQuery.Order != nil && !(*pagingQuery.Order == "asc" || *pagingQuery.Order == "desc") {
			ctx.InvalidParameter("order")
		}

		if pagingQuery.Before != nil && pagingQuery.After != nil {
			ctx.BadRequest("both before and after cursors set")
		}

		ctx.SetPagingQuery(pagingQuery)

		next.ServeHTTP(resp, req)
	})
}
