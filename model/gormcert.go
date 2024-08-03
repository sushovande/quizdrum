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
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GormCert holds a public key from Google
type GormCert struct {
	// ID is the same certificate id used in the JWT signature
	ID string
	// ExpiresMs is when this cert should be fetched again
	ExpiresMs int64
	// CertData is the public key, including the begin and end directives
	CertData string
}

// GetCertWithID returns a parsed usable certificate with the given ID
func (p *Persistence) GetCertWithID(id string) (*rsa.PublicKey, error) {
	allc, err := p.GetUsableCerts()
	if err != nil {
		return nil, err
	}
	for _, c := range allc {
		if c.ID == id {
			r, err := ParseRsaPublicKeyFromPemStr(c.CertData)
			if err != nil {
				return nil, err
			}
			return r, nil
		}
	}
	return nil, fmt.Errorf("no certs matched")
}

// ParseRsaPublicKeyFromPemStr gets a rsa.PublicKey from the string representation
func ParseRsaPublicKeyFromPemStr(pubPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pubPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	certInterface, err := x509.ParseCertificates(block.Bytes)
	if err != nil {
		return nil, err
	}
	pkey := certInterface[0].PublicKey.(*rsa.PublicKey)
	return pkey, nil
}

// GetUsableCerts returns cached certs if still valid, else fresh certs from google server
func (p *Persistence) GetUsableCerts() ([]GormCert, error) {
	ct, err := p.GetAllCerts()
	if err != nil {
		// Although, one might argue we should delete and fetch at this point
		return nil, err
	}
	if len(ct) < 1 || ct[0].ExpiresMs < time.Now().UnixNano()/1000000 {
		p.DeleteAllCerts()
		ct, err = FetchCertFromServer()
		if err != nil {
			return nil, err
		}
		p.InsertCerts(ct) // We are ignoring the error here, best effort caching
		return ct, nil
	}
	return ct, nil
}

// FetchCertFromServer reads from Google and produces fresh certificates
func FetchCertFromServer() ([]GormCert, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v1/certs")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	expHdr := resp.Header["Expires"]
	if len(expHdr) < 1 {
		return nil, fmt.Errorf("did not get an expires header from Google Certs")
	}
	expTime, err := time.Parse("Mon, 2 Jan 2006 15:04:05 MST", expHdr[0])
	if err != nil {
		return nil, err
	}
	expMs := expTime.UnixNano() / 1000000

	x := make(map[string]string)
	if err = json.Unmarshal(body, &x); err != nil {
		return nil, err
	}
	var ct []GormCert
	for k, v := range x {
		ct = append(ct, GormCert{
			ID:        k,
			CertData:  v,
			ExpiresMs: expMs,
		})
	}

	return ct, nil
}

// GetAllCerts gets all the Google certs currently in the db
func (p *Persistence) GetAllCerts() ([]GormCert, error) {
	gc := make([]GormCert, 0)
	if err := p.db.Find(&gc).Error; err != nil {
		return nil, err
	}
	return gc, nil
}

// DeleteAllCerts removes all the certificates cached in the db
func (p *Persistence) DeleteAllCerts() error {
	gc := make([]GormCert, 0)
	return p.db.Delete(&gc, "TRUE").Error
}

// InsertCerts caches all the certificates in the db
func (p *Persistence) InsertCerts(ct []GormCert) error {
	for _, c := range ct {
		if err := p.db.Create(&c).Error; err != nil {
			return err
		}
	}
	return nil
}
