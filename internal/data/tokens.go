package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"time"

	"github.com/BunnyTheLifeguard/greenlight/internal/validator"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Constants for token scope
const (
	ScopeActivation = "activation"
)

// Token struct holds data for individual tokens
type Token struct {
	OID       primitive.ObjectID `bson:"_id"`
	Plaintext string
	Hash      []byte             `bson:"hash"`
	UserID    primitive.ObjectID `bson:"user_id"`
	Expiry    time.Time          `bson:"expiry"`
	Scope     string             `bson:"scope"`
}

// TokenModel type
type TokenModel struct {
	Collection *mongo.Collection
}

func generateToken(userID string, ttl time.Duration, scope string) (*Token, error) {
	uoid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	// New Token instance containing userID, expiry & scope info
	token := &Token{
		UserID: uoid,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	randomBytes := make([]byte, 16)

	// Fill byte slice with random bytes
	_, err = rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	// Encode byte slice to base-32-encoded string & assign to token plaintext
	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)

	// Generate SHA-256 hash of plaintext string & convert to slice
	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token, nil
}

// ValidateTokenPlaintext checks
func ValidateTokenPlaintext(v *validator.Validator, tokenPlaintext string) {
	v.Check(tokenPlaintext != "", "token", "must be provided")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 26 bytes long")
}

// New shortcut method to create a Token struct & add data to DB
func (m TokenModel) New(userID string, ttl time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}

	err = m.Insert(token)
	return token, err
}

// Insert adds specific token to DB
func (m TokenModel) Insert(token *Token) error {
	oid := primitive.NewObjectID()

	args := Token{
		OID:    oid,
		Hash:   token.Hash,
		UserID: token.UserID,
		Expiry: token.Expiry,
		Scope:  token.Scope,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := m.Collection.InsertOne(ctx, args)
	if err != nil {
		return err
	}

	return nil
}

// Get method for userID via token
func (m TokenModel) Get(tokenScope, tokenPlaintext string) (string, error) {
	var result *Token
	// Calculate SHA-256 hash of plaintext token provided by client
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))

	filter := bson.M{
		"hash":  tokenHash[:],
		"scope": tokenScope,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := m.Collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		return "", err
	}

	return result.UserID.Hex(), nil
}

// DeleteAllForUser removes all tokens for specific user & scope
func (m TokenModel) DeleteAllForUser(scope, userID string) error {
	uoid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	delete := bson.M{"user_id": uoid, "scope": scope}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, _ := m.Collection.DeleteOne(ctx, delete)
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}
