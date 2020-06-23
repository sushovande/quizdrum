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

package controller

import (
	"bytes"
	"net/http/httptest"
	"quizdrum/model"
	"quizdrum/view"
	"testing"
)

func TestGuestLogin(t *testing.T) {
	var p model.Persistence
	p.Initialize(":memory:", "oauth_client_fake_id")
	var v view.View
	v.Initialize()
	c := Controller{
		P: &p, V: &v,
	}

	var by []byte
	a := bytes.NewReader(by)
	req := httptest.NewRequest("POST", "/api/common/guest-login", a)
	resp := httptest.NewRecorder()
	c.HandleGuestLogin(resp, req)
	if r := resp.Body.String(); r != "1" {
		t.Errorf("unexpected user id response. want %v, got %v", "1", r)
	}
}
