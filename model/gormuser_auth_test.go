// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import (
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

func TestOAuthLoginValidation(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	sk := jose.SigningKey{
		Algorithm: jose.RS256,
		Key:       privateKey,
	}
	sopt := (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "test-kid")
	sig, err := jose.NewSigner(sk, sopt)
	if err != nil {
		t.Fatal(err)
	}

	_, pem := GenCARoot(privateKey)
	clientID := "test-client-id"

	setupPersistence := func() *Persistence {
		p := &Persistence{}
		if err := p.Initialize(":memory:", clientID); err != nil {
			t.Fatal(err)
		}
		gc := GormCert{
			ID:        "test-kid",
			CertData:  string(pem),
			ExpiresMs: time.Now().Add(time.Hour).Unix() * 1000,
		}
		p.InsertCerts([]GormCert{gc})
		return p
	}

	type extraClaims struct {
		EmailVerified bool `json:"email_verified"`
	}

	tests := []struct {
		name    string
		cl      jwt.Claims
		extra   extraClaims
		wantErr bool
		errSub  string
	}{
		{
			name: "Valid Token",
			cl: jwt.Claims{
				Issuer:   "https://accounts.google.com",
				Audience: jwt.Audience{clientID},
				Expiry:   jwt.NewNumericDate(time.Now().Add(time.Hour)),
				Subject:  "user-123",
			},
			extra:   extraClaims{EmailVerified: true},
			wantErr: false,
		},
		{
			name: "Invalid Issuer",
			cl: jwt.Claims{
				Issuer:   "attacker.com",
				Audience: jwt.Audience{clientID},
				Expiry:   jwt.NewNumericDate(time.Now().Add(time.Hour)),
				Subject:  "user-123",
			},
			extra:   extraClaims{EmailVerified: true},
			wantErr: true,
			errSub:  "invalid issuer",
		},
		{
			name: "Expired Token",
			cl: jwt.Claims{
				Issuer:   "https://accounts.google.com",
				Audience: jwt.Audience{clientID},
				Expiry:   jwt.NewNumericDate(time.Now().Add(-time.Hour)),
				Subject:  "user-123",
			},
			extra:   extraClaims{EmailVerified: true},
			wantErr: true,
			errSub:  "token expired",
		},
		{
			name: "Wrong Audience",
			cl: jwt.Claims{
				Issuer:   "https://accounts.google.com",
				Audience: jwt.Audience{"wrong-client-id"},
				Expiry:   jwt.NewNumericDate(time.Now().Add(time.Hour)),
				Subject:  "user-123",
			},
			extra:   extraClaims{EmailVerified: true},
			wantErr: true,
			errSub:  "wrong client ID",
		},
		{
			name: "Email Not Verified",
			cl: jwt.Claims{
				Issuer:   "https://accounts.google.com",
				Audience: jwt.Audience{clientID},
				Expiry:   jwt.NewNumericDate(time.Now().Add(time.Hour)),
				Subject:  "user-123",
			},
			extra:   extraClaims{EmailVerified: false},
			wantErr: true,
			errSub:  "email not verified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := setupPersistence()
			raw, err := jwt.Signed(sig).Claims(tt.cl).Claims(tt.extra).Serialize()
			if err != nil {
				t.Fatal(err)
			}

			_, err = p.OAuthLoginFromToken(raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("OAuthLoginFromToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if tt.errSub != "" && !strings.Contains(err.Error(), tt.errSub) {
					t.Errorf("OAuthLoginFromToken() error = %v, want error containing %v", err, tt.errSub)
				}
			}
		})
	}
}
