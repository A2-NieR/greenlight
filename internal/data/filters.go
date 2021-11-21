package data

import (
	"github.com/BunnyTheLifeguard/greenlight/internal/validator"
)

// Filters holds url query values
type Filters struct {
	Page         int
	PageSize     int
	Sort         string
	SortSafelist []string
}

// Metadata holds pagination info
type Metadata struct {
	CurrentPage  int `json:"current_page,omitempty"`
	PageSize     int `json:"page_size,omitempty"`
	TotalRecords int `json:"total_records,omitempty"`
}

// ValidateFilters validates query values
func ValidateFilters(v *validator.Validator, f Filters) {
	v.Check(f.Page <= 10_000_000, "page", "must be a maximum of 10 million")
	v.Check(f.PageSize <= 100, "page_size", "must be a maximum of 100")

	v.Check(validator.In(f.Sort, f.SortSafelist...), "sort", "invalid sort value")
}

func (f Filters) limit() int {
	return f.PageSize
}

func (f Filters) offset() int {
	if f.Page == 0 {
		return 0
	}
	return (f.Page - 1) * f.PageSize
}

func calculateMetadata(totalRecords, page, pageSize int) Metadata {
	if totalRecords == 0 {
		return Metadata{}
	}

	return Metadata{
		CurrentPage:  page,
		PageSize:     pageSize,
		TotalRecords: totalRecords,
	}
}
