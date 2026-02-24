package main

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	// Read JWT secret from environment
	secret := os.Getenv("SUPABASE_JWT_SECRET")
	supabaseURL := os.Getenv("SUPABASE_URL")
	if secret == "" || supabaseURL == "" {
		fmt.Fprintln(os.Stderr, "Error: SUPABASE_JWT_SECRET and SUPABASE_URL environment variables must be set")
		fmt.Fprintln(os.Stderr, "Usage: SUPABASE_JWT_SECRET=secret SUPABASE_URL=https://xyz.supabase.co go run scripts/generate-jwt.go")
		os.Exit(1)
	}

	// Create claims with Supabase-compatible structure
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":  "test-user-id",
		"role": "authenticated",
		"aud":  "authenticated",
		"iat":  now.Unix(),
		"exp":  now.Add(time.Hour).Unix(),
		"iss":  supabaseURL + "/auth/v1",
	}

	// Create token with HS256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error signing token: %v\n", err)
		os.Exit(1)
	}

	// Print the token
	fmt.Println(tokenString)
}
