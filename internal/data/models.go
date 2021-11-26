package data

import (
	"errors"

	"go.mongodb.org/mongo-driver/mongo"
)

// ErrEditConflict custom error
var (
	ErrEditConflict = errors.New("edit conflict")
)

// Models struct wraps MovieModel
type Models struct {
	Movies MovieModel
	User   UserModel
	Token  TokenModel
}

// NewModels returns Models struct containing initialized MovieModel
func NewModels(data, user, token *mongo.Collection) Models {
	return Models{
		Movies: MovieModel{Collection: data},
		User:   UserModel{Collection: user},
		Token:  TokenModel{Collection: token},
	}
}
