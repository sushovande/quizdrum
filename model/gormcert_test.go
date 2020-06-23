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
	"testing"
)

func TestCertDBOperations(t *testing.T) {
	var p Persistence
	if err := p.Initialize(":memory:", "oauth_client_fake_id"); err != nil {
		t.Fatal(err)
	}

	gc := GormCert{
		ID:        "id1",
		CertData:  "data2",
		ExpiresMs: 123,
	}
	gcs := make([]GormCert, 0)
	gcs = append(gcs, gc)
	if err := p.InsertCerts(gcs); err != nil {
		t.Fatal(err)
	}

	gotcerts, err := p.GetAllCerts()
	if err != nil {
		t.Fatal(err)
	}
	if len(gotcerts) != 1 {
		t.Fatalf("different number of certs. want: %v, got: %v", 1, len(gotcerts))
	}
	if gotcerts[0].ID != gc.ID {
		t.Errorf("wrong id. want: %v, got: %v", gc.ID, gotcerts[0].ID)
	}

	if err := p.DeleteAllCerts(); err != nil {
		t.Errorf("could not delete. %v", err)
	}

	gotcerts, err = p.GetAllCerts()
	if err != nil {
		t.Fatal(err)
	}
	if len(gotcerts) != 0 {
		t.Errorf("unexpected number of certs received. want: 0, got: %v", len(gotcerts))
	}
}
