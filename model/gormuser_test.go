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
	"time"
)

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
