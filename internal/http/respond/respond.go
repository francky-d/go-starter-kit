// Package respond writes HTTP responses in the application's JSON envelope.
//
// Success responses are written by controllers through OK, Created,
// NoContent and Paginated. Error responses are written only by the central
// error handler (see ErrorHandler): controllers return errors instead.
package respond

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Meta holds metadata about the data of a response, such as pagination.
type Meta map[string]any

// Envelope is the body of every successful JSON response.
type Envelope struct {
	Data any  `json:"data"`
	Meta Meta `json:"meta,omitempty"`
}

// Pagination describes one page of a collection.
type Pagination struct {
	Page    int
	PerPage int
	Total   int64
}

// LastPage returns the number of the last page, at least 1.
func (p Pagination) LastPage() int {
	if p.PerPage <= 0 || p.Total <= 0 {
		return 1
	}

	return int((p.Total + int64(p.PerPage) - 1) / int64(p.PerPage))
}

// OK answers 200 with data.
func OK(c echo.Context, data any) error {
	return c.JSON(http.StatusOK, Envelope{Data: data})
}

// Created answers 201 with the created resource.
func Created(c echo.Context, data any) error {
	return c.JSON(http.StatusCreated, Envelope{Data: data})
}

// NoContent answers 204 without a body.
func NoContent(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

// Paginated answers 200 with one page of a collection and its pagination
// metadata.
func Paginated(c echo.Context, data any, page Pagination) error {
	return c.JSON(http.StatusOK, Envelope{
		Data: data,
		Meta: Meta{
			"page":      page.Page,
			"per_page":  page.PerPage,
			"total":     page.Total,
			"last_page": page.LastPage(),
		},
	})
}
