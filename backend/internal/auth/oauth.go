package auth

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
)

type TokenStore struct {
	rdb *redis.Client
}

func NewTokenStore(rdb *redis.Client) *TokenStore {
	return &TokenStore{rdb: rdb}
}

// Save serializes the entire token object to JSON and saves it in Redis.
func (t *TokenStore) Save(ctx context.Context, userID string, tok *oauth2.Token) error {
	// Serialize token to JSON
	tokenJSON, err := json.Marshal(tok)
	if err != nil {
		return err
	}
	// Save the JSON string to Redis with the original expiry
	return t.rdb.Set(ctx, "token:"+userID, tokenJSON, time.Until(tok.Expiry)).Err()
}

// Get retrieves the token from Redis and deserializes it from JSON.
func (t *TokenStore) Get(ctx context.Context, userID string) (*oauth2.Token, error) {
	// Get the JSON string from Redis
	tokenJSON, err := t.rdb.Get(ctx, "token:"+userID).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Return nil, nil if token not found
		}
		return nil, err
	}

	// Deserialize JSON back to a token object
	var tok oauth2.Token
	if err := json.Unmarshal([]byte(tokenJSON), &tok); err != nil {
		return nil, err
	}

	return &tok, nil
}
