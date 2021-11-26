package data

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/BunnyTheLifeguard/greenlight/internal/validator"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

// Custom ErrDuplicateEmail error
var (
	ErrDuplicateName  = errors.New("duplicate name")
	ErrDuplicateEmail = errors.New("duplicate email")
)

// User represents individual user, pw & version excluded from res
type User struct {
	OID       primitive.ObjectID `json:"-" bson:"_id"`
	ID        string             `json:"-" bson:"id"`
	CreatedAt time.Time          `json:"-" bson:"created_at"`
	Name      string             `json:"name" bson:"name"`
	Email     string             `json:"email" bson:"email"`
	Password  password           `json:"-" bson:"password"`
	Activated bool               `json:"activated" bson:"activated"`
	Version   int                `json:"-" bson:"version"`
}

// Pointer to string to distinguish between pw not present & empty string ""
type password struct {
	plaintext *string
	hash      []byte
}

// UserModel wraps connection pool
type UserModel struct {
	Collection *mongo.Collection
}

func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return err
	}

	p.plaintext = &plaintextPassword
	p.hash = hash

	return nil
}

func (p *password) Matches(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}

	return true, nil
}

// ValidateEmail checks for empty & regex pattern
func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

// ValidatePasswordPlaintext checks for empty & min/max pw length
func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")
}

// ValidateUser checks for empty username & length, calls other validators
func ValidateUser(v *validator.Validator, user *User) {
	v.Check(user.Name != "", "name", "must be provided")
	v.Check(len(user.Name) <= 500, "name", "must be more than 500 bytes long")

	ValidateEmail(v, user.Email)

	if user.Password.plaintext != nil {
		ValidatePasswordPlaintext(v, *user.Password.plaintext)
	}

	if user.Password.hash == nil {
		panic("missing password hash for user")
	}
}

// Insert method to create a new user
func (m UserModel) Insert(user *User) (string, error) {
	oid := primitive.NewObjectID()
	id := oid.Hex()

	args := User{
		OID:       oid,
		ID:        id,
		CreatedAt: time.Now(),
		Name:      user.Name,
		Email:     user.Email,
		Password:  user.Password,
		Activated: user.Activated,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := m.Collection.InsertOne(ctx, args)
	if err != nil {
		mongoErr := err.(mongo.WriteException)
		if strings.Contains(mongoErr.WriteErrors[0].Message, "name") {
			return "", ErrDuplicateName
		} else if strings.Contains(mongoErr.WriteErrors[0].Message, "email") {
			return "", ErrDuplicateEmail
		}
		return "", err
	}

	filter := bson.M{"_id": oid}
	update := bson.M{"$inc": bson.M{"version": 1}}
	_ = m.Collection.FindOneAndUpdate(ctx, filter, update)
	return id, nil
}

// GetByEmail method to get details of specific user
func (m UserModel) GetByEmail(email string) (*User, error) {
	var result *User

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"email": email}
	err := m.Collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Update method for editing user's details
func (m UserModel) Update(user *User, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"name":      user.Name,
			"email":     user.Email,
			"password":  user.Password,
			"activated": user.Activated,
		}, "$inc": bson.M{"version": 1},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = m.Collection.UpdateByID(ctx, oid, update)
	if err != nil {
		switch {
		case mongo.IsDuplicateKeyError(err):
			return ErrDuplicateEmail
		case err == mongo.ErrNoDocuments:
			return ErrEditConflict
		default:
			return err
		}

	}

	return nil
}
