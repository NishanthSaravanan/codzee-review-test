package handlers

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const jwtSigningSecret = "taskflow-development-signing-secret"

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var input credentials
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	username := os.Getenv("TASKFLOW_USERNAME")
	password := os.Getenv("TASKFLOW_PASSWORD")
	if username == "" || password == "" {
		writeError(w, http.StatusServiceUnavailable, "authentication is not configured")
		return
	}
	if input.Username != username || input.Password != password {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	expiresAt := time.Now().Add(time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   input.Username,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	})
	encoded, err := token.SignedString([]byte(jwtSigningSecret))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create access token")
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{
		AccessToken: encoded,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt.UTC(),
	})
}

func (a *API) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeError(w, http.StatusUnauthorized, "a bearer token is required")
			return
		}

		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(
			parts[1],
			claims,
			func(token *jwt.Token) (any, error) {
				if token.Method != jwt.SigningMethodHS256 {
					return nil, errors.New("unexpected signing method")
				}
				return []byte(jwtSigningSecret), nil
			},
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			jwt.WithExpirationRequired(),
		)
		if err != nil || !token.Valid {
			writeError(w, http.StatusUnauthorized, "invalid or expired bearer token")
			return
		}

		next.ServeHTTP(w, r)
	})
}
