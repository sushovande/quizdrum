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
	"log"
	"net/http"
	"quizdrum/model"
	"quizdrum/view"
	"time"

	"google.golang.org/protobuf/proto"
)

// RenderHomepage is the UI handler that shows the homepage
func (c *Controller) RenderHomepage(w http.ResponseWriter, r *http.Request) {
	u, err := c.P.GetUserFromCookieAndError(r.Cookie("sid"))
	if err != nil {
		u = &model.User{
			Id: proto.Int64(-1),
		}
	}
	qs, err := c.P.GetAllQuizzes()
	if view.Should500(err, w, "could not fetch quizzes") {
		return
	}

	type DisplayQuiz struct {
		ID          int64
		Title       string
		Description string
		CannotWrite bool
	}
	dq := make([]DisplayQuiz, 0)

	for _, qz := range qs {
		dq = append(dq, DisplayQuiz{
			ID:          qz.GetId(),
			Title:       qz.GetTitle(),
			Description: qz.GetHtmlDescription(),
			CannotWrite: !matchesSomeQuizmaster(qz, u),
		})
	}

	s := struct {
		QM []DisplayQuiz
		U  *model.User
	}{
		QM: dq,
		U:  u,
	}

	c.V.RenderTemplate(w, "index.html", s)
}

// RenderLogin is the UI handler that shows the login page.
func (c *Controller) RenderLogin(w http.ResponseWriter, r *http.Request) {
	d := struct {
		OAuthClientID string
	}{
		OAuthClientID: c.P.OAuthClientID,
	}
	c.V.RenderTemplate(w, "login.html", d)
}

// HandleLogout is the API handler that deletes the cookie from the browser and the db
func (c *Controller) HandleLogout(w http.ResponseWriter, r *http.Request) {
	expiredCookie := http.Cookie{
		Name:     "sid",
		Value:    "",
		Expires:  time.Now().Add(-24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, &expiredCookie)
	d := struct {
		OAuthClientID string
	}{
		OAuthClientID: c.P.OAuthClientID,
	}
	c.V.RenderTemplate(w, "logout.html", d)

	// Removing the cookie from the db is best-effort
	cko, err := r.Cookie("sid")
	if err != nil {
		log.Printf("did not find a cookie to begin with: %v\n", err)
		return
	}

	err = c.P.DeleteCookie(cko.Value)
	if err != nil {
		log.Printf("could not remove the cookie from the db: %v\n", err)
	}
}

func matchesSomeQuizmaster(qz *model.Quiz, u *model.User) bool {
	for _, qm := range qz.GetQuizmasters() {
		if model.UserMatches(qm, u) {
			return true
		}
	}
	return false
}
