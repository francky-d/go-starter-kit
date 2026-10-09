// Package respond writes responses in the JSON envelope. Controllers use the
// success helpers; error bodies are written only by ErrorHandler.
package respond

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type Meta map[string]any

type Envelope struct {
	Data any  `json:"data"`
	Meta Meta `json:"meta,omitempty"`
}

type Pagination struct {
	Page    int
	PerPage int
	Total   int64
}

// LastPage is at least 1, even for an empty collection.
func (p Pagination) LastPage() int {
	if p.PerPage <= 0 || p.Total <= 0 {
		return 1
	}

	return int((p.Total + int64(p.PerPage) - 1) / int64(p.PerPage))
}

func OK(c echo.Context, data any) error {
	return c.JSON(http.StatusOK, Envelope{Data: data})
}

func Created(c echo.Context, data any) error {
	return c.JSON(http.StatusCreated, Envelope{Data: data})
}

func NoContent(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func Paginated(c echo.Context, data any, page Pagination) error {
	return c.JSON(http.StatusOK, Envelope{Data: data, Meta: page.meta()})
}

func (p Pagination) meta() Meta {
	return Meta{
		"page":      p.Page,
		"per_page":  p.PerPage,
		"total":     p.Total,
		"last_page": p.LastPage(),
	}
}
