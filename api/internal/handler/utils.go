package handler

import (
	"math"
	"net/http"
	"strconv"
)

// ListParams holds standard pagination, sorting, and search fields.
type ListParams struct {
	Page    int
	PerPage int
	SortBy  string
	SortDir string // "asc" or "desc"
	Search  string
}

// Offset returns the SQL offset.
func (p ListParams) Offset() int64 {
	return int64((p.Page - 1) * p.PerPage)
}

// Limit returns the SQL limit.
func (p ListParams) Limit() int64 {
	return int64(p.PerPage)
}

// ParseListParams extracts common list parameters from the request query.
func ParseListParams(r *http.Request) ListParams {
	q := r.URL.Query()

	page := 1
	if p, err := strconv.Atoi(q.Get("page")); err == nil && p > 0 {
		page = p
	}

	perPage := 50
	if pp, err := strconv.Atoi(q.Get("per_page")); err == nil && pp > 0 {
		if pp > 100 {
			perPage = 100
		} else {
			perPage = pp
		}
	}

	sortBy := q.Get("sort_by")
	if sortBy == "" {
		sortBy = "created_at" // default fallback
	}

	sortDir := q.Get("sort_dir")
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc" // default
	}

	return ListParams{
		Page:    page,
		PerPage: perPage,
		SortBy:  sortBy,
		SortDir: sortDir,
		Search:  q.Get("search"),
	}
}

// Meta wraps pagination metadata for list responses.
type Meta struct {
	Total      int `json:"total"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
}

// NewMeta initializes a Meta object based on total items.
func NewMeta(total, page, perPage int) Meta {
	var totalPages int
	if perPage > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(perPage)))
	}
	return Meta{
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}
}
