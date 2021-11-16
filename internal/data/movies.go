package data

import (
	"time"

	"github.com/BunnyTheLifeguard/greenlight/internal/validator"
	"go.mongodb.org/mongo-driver/mongo"
)

// Movie struct
type Movie struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"-"`
	Title     string    `json:"title"`
	Year      int32     `json:"year,omitempty"`
	Runtime   Runtime   `json:"runtime,omitempty"`
	Genres    []string  `json:"genres,omitempty"`
	Version   int32     `json:"version"`
}

// MovieModel struct type wraps a MongoDB collection
type MovieModel struct {
	Collection *mongo.Collection
}

// ValidateMovie check for valid JSON
func ValidateMovie(v *validator.Validator, movie *Movie) {
	v.Check(movie.Title != "", "title", "must be provided")
	v.Check(len(movie.Title) <= 500, "title", "must not be more than 500 bytes long")

	v.Check(movie.Year != 0, "year", "must be provided")
	v.Check(movie.Year >= 1888, "year", "must be greater than 1888")
	v.Check(movie.Year <= int32(time.Now().Year()), "year", "must not be in the future")

	v.Check(movie.Runtime != 0, "runtime", "must be provided")
	v.Check(movie.Runtime > 0, "runtime", "must be a positive integer")

	v.Check(movie.Genres != nil, "genres", "must be provided")
	v.Check(len(movie.Genres) >= 1, "genres", "must contain at least 1 genre")
	v.Check(len(movie.Genres) <= 5, "genres", "must not contain more than 5 genres")
	v.Check(validator.Unique(movie.Genres), "genres", "must not contain duplicate values")
}

// Insert placeholder method for inserting a new record
func (m MovieModel) Insert(movie *Movie) error {
	return nil
}

// Get placeholder method for fetching a specific record
func (m MovieModel) Get(id string) (*Movie, error) {
	return nil, nil
}

// Update placeholder method for updating a specific record
func (m MovieModel) Update(movie *Movie) error {
	return nil
}

// Delete placeholder method for deleting a specific record
func (m MovieModel) Delete(id string) error {
	return nil
}
