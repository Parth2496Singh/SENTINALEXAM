// Package keycloak provides JWT validation against Keycloak's JWKS endpoint.
package keycloak

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sentinelexam/backend/internal/assessment/service"
	"github.com/sentinelexam/backend/internal/platform/config"
)

// Authenticator validates JWT tokens from Keycloak.
type Authenticator struct {
	cfg    *config.Config
	logger *slog.Logger
}

func NewAuthenticator(cfg *config.Config, logger *slog.Logger) *Authenticator {
	return &Authenticator{cfg: cfg, logger: logger}
}

type keycloakClaims struct {
	jwt.RegisteredClaims
	Email             string     `json:"email"`
	PreferredUsername string     `json:"preferred_username"`
	Name              string     `json:"name"`
	RealmAccess       realmAccess `json:"realm_access"`
}

type realmAccess struct {
	Roles []string `json:"roles"`
}

// ParseClaims extracts and validates the bearer token.
// In development mode, accepts "dev:<roles>:<sub>" tokens for testing.
func (a *Authenticator) ParseClaims(ctx context.Context, authHeader string) (*service.AuthClaims, error) {
	if authHeader == "" {
		return nil, fmt.Errorf("missing authorization header")
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return nil, fmt.Errorf("malformed authorization header")
	}
	rawToken := parts[1]

	if a.cfg.Environment == "development" && strings.HasPrefix(rawToken, "dev:") {
		return parseDevToken(rawToken)
	}
	return a.parseJWT(ctx, rawToken)
}

// parseDevToken handles the development bypass token format: "dev:<roles_csv>:<sub_uuid>"
func parseDevToken(raw string) (*service.AuthClaims, error) {
	body := raw[4:]
	parts := strings.SplitN(body, ":", 2)
	roles := strings.Split(parts[0], ",")
	sub := "00000000-0000-0000-0000-000000000001"
	if len(parts) > 1 && parts[1] != "" {
		sub = parts[1]
	}
	subID, err := uuid.Parse(sub)
	if err != nil {
		return nil, fmt.Errorf("dev token: invalid sub uuid: %w", err)
	}
	return &service.AuthClaims{
		Sub:         sub,
		KeycloakSub: subID,
		Email:       fmt.Sprintf("dev_%s@sentinel.local", sub[:8]),
		DisplayName: "Dev User",
		Roles:       roles,
	}, nil
}

func (a *Authenticator) parseJWT(_ context.Context, rawToken string) (*service.AuthClaims, error) {
	token, err := jwt.ParseWithClaims(rawToken, &keycloakClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			kid, _ := token.Header["kid"].(string)
			return a.fetchJWKSKey(kid)
		},
		jwt.WithIssuedAt(),
		jwt.WithIssuer(a.cfg.KeycloakIssuer),
	)
	if err != nil {
		return nil, fmt.Errorf("jwt validation failed: %w", err)
	}
	claims, ok := token.Claims.(*keycloakClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	subID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, fmt.Errorf("invalid sub claim: %w", err)
	}
	displayName := claims.Name
	if displayName == "" {
		displayName = claims.PreferredUsername
	}
	return &service.AuthClaims{
		Sub:         claims.Subject,
		KeycloakSub: subID,
		Email:       claims.Email,
		DisplayName: displayName,
		Roles:       claims.RealmAccess.Roles,
	}, nil
}

type jwksResponse struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (a *Authenticator) fetchJWKSKey(kid string) (*rsa.PublicKey, error) {
	jwksURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/certs",
		a.cfg.KeycloakURL, a.cfg.KeycloakRealm)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode jwks: %w", err)
	}
	for _, key := range jwks.Keys {
		if key.Kid == kid {
			return rsaPublicKeyFromJWK(key)
		}
	}
	return nil, fmt.Errorf("jwks key not found for kid: %s", kid)
}

func rsaPublicKeyFromJWK(key jwk) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}
	// Pad eBytes to 4 bytes
	if len(eBytes) < 4 {
		padded := make([]byte, 4)
		copy(padded[4-len(eBytes):], eBytes)
		eBytes = padded
	}
	e := int(binary.BigEndian.Uint32(eBytes))
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: e,
	}, nil
}
