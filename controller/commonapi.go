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
	"fmt"
	"net/http"
	"quizdrum/view"
	"time"

	"github.com/google/uuid"
)

var (
	guestCookieValidity = 30 * 24 * time.Hour
	oauthCookieValidity = 60 * 24 * time.Hour
)

// HandleGuestLogin is the API handler that sets a creates a guest account and sets the cookie
func (c *Controller) HandleGuestLogin(w http.ResponseWriter, r *http.Request) {
	ckuuid, err := uuid.NewRandom()
	if view.Should500(err, w, "could not generate a cookie") {
		return
	}
	ck := ckuuid.String()
	expiry := time.Now().Add(guestCookieValidity)
	uid, err := c.P.NewGuestLogin(ck, expiry.Unix())
	if view.Should500(err, w, "could not log in as guest") {
		return
	}

	cookie := http.Cookie{
		Name:     "sid",
		Value:    ck,
		Expires:  expiry,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, &cookie)
	view.WriteJSONString(w, fmt.Sprint(uid))
}

// HandleOauthLogin is the API handler that sets a creates a guest account and sets the cookie
func (c *Controller) HandleOauthLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if view.Should500(err, w, "error reading idToken from request") {
		return
	}
	idToken := r.PostFormValue("idtoken")
	if len(idToken) == 0 {
		view.Should500(fmt.Errorf("idtoken empty or not found"), w, "idtoken empty or not found")
		return
	}
	u, err := c.P.OAuthLoginFromToken(idToken)
	if view.Should500(err, w, "oauth login failed") {
		return
	}
	ckuuid, err := uuid.NewRandom()
	if view.Should500(err, w, "could not generate a cookie") {
		return
	}
	expiry := time.Now().Add(oauthCookieValidity)
	ck := ckuuid.String()
	err = c.P.NewCookieForUser(ck, uint(u.GetId()), expiry.Unix())
	if view.Should500(err, w, "could not store the cookie") {
		return
	}
	cookie := http.Cookie{
		Name:     "sid",
		Value:    ck,
		Expires:  expiry,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, &cookie)
	view.WriteJSONString(w, fmt.Sprint(u.GetId()))
}
