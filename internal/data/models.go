package data

import "go.mongodb.org/mongo-driver/mongo"

// Models struct wraps MovieModel
type Models struct {
	Movies MovieModel
}

// NewModels returns Models struct containing initialized MovieModel
func NewModels(coll *mongo.Collection) Models {
	return Models{
		Movies: MovieModel{Collection: coll},
	}
}
