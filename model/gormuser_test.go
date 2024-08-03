// Copyright 2020 Google LLC
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
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

func GenCARoot(pk *rsa.PrivateKey) (*x509.Certificate, []byte) {
	rootTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1000),
		Subject: pkix.Name{
			Country:      []string{"IN"},
			Organization: []string{"Example Ltd."},
			CommonName:   "Root CA",
		},
		NotBefore:             time.Now().Add(time.Hour * time.Duration(-480)),
		NotAfter:              time.Now().Add(time.Hour * time.Duration(960)),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            2,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &rootTemplate, &rootTemplate, &pk.PublicKey, pk)
	if err != nil {
		panic("Failed to create certificate:" + err.Error())
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		panic("Failed to parse certificate:" + err.Error())
	}

	b := pem.Block{Type: "CERTIFICATE", Bytes: certBytes}
	certPEM := pem.EncodeToMemory(&b)
	return cert, certPEM
}

func TestCreateGuestUser(t *testing.T) {
	var p Persistence
	if err := p.Initialize(":memory:", "oauth_key_fake_id"); err != nil {
		t.Fatal(err)
	}
	expires := time.Now().Add(36 * time.Hour).Unix()
	uid, err := p.NewGuestLogin("cookiecookie", expires)
	if err != nil {
		t.Fatal(err)
	}
	if uid == 0 {
		t.Errorf("The user id should not be zero.")
	}
	u, err := p.GetUserFromCookie("cookiecookie")
	if err != nil {
		t.Fatal(err)
	}
	if u.GetId() != int64(uid) {
		t.Errorf("mismatched user ID. want %v got %v", uid, u.GetId())
	}
	if u.GetGoogleUser().GetSub() != "" {
		t.Errorf("unexpected google sub claim. want <empty string> got %v", u.GetGoogleUser().GetSub())
	}
}

func TestWithJwt(t *testing.T) {
	// Create a dummy key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	// Create a JWT using that key and some dummy data.
	sk := jose.SigningKey{
		Algorithm: jose.RS256,
		Key:       privateKey,
	}
	sopt := (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "kid-1")
	sig, err := jose.NewSigner(sk, sopt)
	if err != nil {
		t.Error(err)
	}
	cl := jwt.Claims{
		Issuer:    "https://accounts.google.com",
		Audience:  jwt.Audience{"client-id-1.apps.googleusercontent.com"},
		Subject:   "121212",
		NotBefore: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(-480))),
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(-360))),
		Expiry:    jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(360))),
		ID:        "jwt-id-1",
	}
	cl2 := struct {
		Azp           string `json:"azp,omitempty"`
		Email         string `json:"email,omitempty"`
		EmailVerified bool   `json:"email_verified,omitempty"`
		Name          string `json:"name,omitempty"`
		Picture       string `json:"picture,omitempty"`
		GivenName     string `json:"given_name,omitempty"`
		FamilyName    string `json:"family_name,omitempty"`
	}{
		Azp:           "client-id-1.apps.googleusercontent.com",
		Email:         "person@example.com",
		EmailVerified: true,
		Name:          "People Person",
		Picture:       "image-url-here",
		GivenName:     "People",
		FamilyName:    "Person",
	}
	// The string 'raw' now contains the encoded token-id that we typically get from Google.
	raw, err := jwt.Signed(sig).Claims(cl).Claims(cl2).Serialize()
	if err != nil {
		t.Error(err)
	}

	// Set up our DB with the public key we just generated.
	var p Persistence
	if err := p.Initialize(":memory:", "client-id-1.apps.googleusercontent.com"); err != nil {
		t.Fatal(err)
	}
	p.OAuthClientID = "client-id-1.apps.googleusercontent.com"

	_, pem := GenCARoot(privateKey)
	var gc GormCert
	gc.CertData = string(pem)
	gc.ID = "kid-1"
	gc.ExpiresMs = time.Now().Add(960*time.Hour).Unix() * 1000
	if err := p.InsertCerts([]GormCert{gc}); err != nil {
		t.Error(err)
	}

	// Simulate a login event using the JWT id token above
	u, err := p.OAuthLoginFromToken(raw)
	if err != nil {
		t.Error(err)
	}
	if u.GetGoogleUser().GetEmail() != "person@example.com" {
		t.Errorf("wrong email. want: person@example.com, got: %v", u.GetGoogleUser().GetEmail())
	}
	if u.GetGoogleUser().GetPicture() != "image-url-here" {
		t.Errorf("wrong image url. want: image-url-here, got: %v", u.GetGoogleUser().GetEmail())
	}
}
