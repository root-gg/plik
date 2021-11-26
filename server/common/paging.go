package common

import (
	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"
	"github.com/root-gg/utils"
)

// PagingQuery for the paging system
type PagingQuery struct {
	Before *string `json:"before"`
	After  *string `json:"after"`
	Limit  *int    `json:"limit"`
	Order  *string `json:"order"`
}

// NewPagingQuery return a new paging query
func NewPagingQuery() (pq *PagingQuery) {
	return &PagingQuery{}
}

// WithLimit set the paging query limit if limit is a valid positive integer
func (pq *PagingQuery) WithLimit(limit int) *PagingQuery {
	pq.Limit = &limit
	return pq
}

// WithOrder set the paging query order if oder is a valid order string "asc" or "desc"
func (pq *PagingQuery) WithOrder(order string) *PagingQuery {
	pq.Order = &order
	return pq
}

// WithBeforeCursor set the paging query before cursor and unset the after cursor
func (pq *PagingQuery) WithBeforeCursor(cursor string) *PagingQuery {
	pq.Before = &cursor
	return pq
}

// WithAfterCursor set the paging query after cursor and unset the before cursor
func (pq *PagingQuery) WithAfterCursor(cursor string) *PagingQuery {
	pq.After = &cursor
	return pq
}

// Paginator return a new Paginator for the PagingQuery
func (pq *PagingQuery) Paginator() *paginator.Paginator {
	p := paginator.New()

	if pq.After != nil {
		p.SetAfterCursor(*pq.After) // [default: nil]
	}

	if pq.Before != nil {
		p.SetBeforeCursor(*pq.Before) // [default: nil]
	}

	if pq.Limit != nil {
		p.SetLimit(*pq.Limit) // [default: 10]
	}

	if pq.Order != nil && *pq.Order == "asc" {
		p.SetOrder(paginator.ASC) // [default: paginator.DESC]
	}

	return p
}

// PagingResponse for the paging system
type PagingResponse struct {
	After   *string       `json:"after"`
	Before  *string       `json:"before"`
	Results []interface{} `json:"results"`
}

// NewPagingResponse create a new PagingResponse from query results ( results must be a slice )
func NewPagingResponse(results interface{}, cursor *paginator.Cursor) (pr *PagingResponse) {
	pr = &PagingResponse{}
	pr.Results = utils.ToInterfaceArray(results)
	pr.Before = cursor.Before
	pr.After = cursor.After
	return pr
}
