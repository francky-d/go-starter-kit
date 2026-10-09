package repositories

import "gorm.io/gorm"

const (
	DefaultPerPage = 20
	MaxPerPage     = 100
)

// NormalizePage returns a page of at least 1 and a page size defaulted to
// DefaultPerPage and capped at MaxPerPage, so a client cannot ask for a
// whole table at once.
func NormalizePage(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}

	if perPage < 1 {
		perPage = DefaultPerPage
	}

	return page, min(perPage, MaxPerPage)
}

func Paginate(page, perPage int) func(*gorm.DB) *gorm.DB {
	page, perPage = NormalizePage(page, perPage)

	return func(db *gorm.DB) *gorm.DB {
		return db.Offset((page - 1) * perPage).Limit(perPage)
	}
}
