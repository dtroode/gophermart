package auth

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWT_GetUserID(t *testing.T) {
	secretKey := "a-string-secret-at-least-256-bits-long"

	tests := map[string]struct {
		tokenString      string
		expectedResponse uuid.UUID
		wantError        bool
	}{
		"wrong signing method": {
			tokenString:      "eyJhbGciOiJIUzM4NCIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.owv7q9nVbW5tqUezF_G2nHTra-ANW3HqW9epyVwh08Y-Z-FKsnG8eBIpC4GTfTVU",
			expectedResponse: uuid.Nil,
			wantError:        true,
		},
		"failed to parse token": {
			tokenString:      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYSJ9.1AFwgCCUaVF6OP6TTsbGg9Ro26kd7VrDsbmYH3uA_uc",
			expectedResponse: uuid.Nil,
			wantError:        true,
		},
		"token is invalid": {
			tokenString:      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYTJmMTY0ODAtZWU3MS00ODlmLWI0NTctMjdkOWZiZTZjZDhhIn0.0HSlnFqLqqJ6xXhqisARVvgrScVdxfj6lYpzAzuX5R4",
			expectedResponse: uuid.Nil,
			wantError:        true,
		},
		"user id is nil": {
			tokenString:      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.KMUFsIDTnFmyG3nMiGM6H9FNFUROf3wh7SmqJp-QV30",
			expectedResponse: uuid.Nil,
		},
		"user id is not nil": {
			tokenString:      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYTJmMTY0ODAtZWU3MS00ODlmLWI0NTctMjdkOWZiZTZjZDhhIn0.MEW-5jJPEr_IND4PBAnYzj_XMhrfQCsZjcx2JTGlpZg",
			expectedResponse: uuid.MustParse("a2f16480-ee71-489f-b457-27d9fbe6cd8a"),
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			t.Parallel()

			j := NewJWT(secretKey)

			userID, err := j.GetUserID(context.Background(), tt.tokenString)

			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.expectedResponse, userID)
		})
	}
}

func TestJWT_CreateToken(t *testing.T) {
	secretKey := "a-string-secret-at-least-256-bits-long"
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		j := NewJWT(secretKey)

		tokenString, err := j.CreateToken(context.Background(), userID)

		require.NoError(t, err)
		require.NotEqual(t, "", tokenString)

		claims := Claims{}
		_, err = jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
			_, ok := token.Method.(*jwt.SigningMethodHMAC)
			assert.True(t, ok)

			return []byte(j.secretKey), nil
		})
		require.NoError(t, err)

		assert.NotNil(t, claims.RegisteredClaims.IssuedAt)
		assert.Equal(t, userID, claims.UserID)
	})
}
