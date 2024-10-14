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

package view

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
)

var (
	templateDir = "view/templates"
)

// View holds all the data needed to render the information to the user
type View struct {
	tm *template.Template
}

// Initialize loads the templates and prepares the view to render
func (v *View) Initialize() error {
	t, err := loadAllTemplateFiles()
	if err != nil {
		return err
	}
	v.tm = t
	return nil
}

// RenderTemplate takes in the template pool, name, and params, does error checking, and renders.
func (v *View) RenderTemplate(w http.ResponseWriter, nm string, data interface{}) {
	err := v.tm.ExecuteTemplate(w, nm, data)
	if Should500(err, w, "HTTP 500: We could not render the page: "+nm) {
		return
	}
}

// Should500 checks if err is not nil, and if so logs and sends the error
func Should500(err error, w http.ResponseWriter, msg string) bool {
	if err != nil {
		log.Printf("page-err: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, msg)
		return true
	}
	return false
}

// UnauthIfError checks if err is not nil, and if so logs and sends the error
func UnauthIfError(err error, w http.ResponseWriter, msg string) bool {
	if err != nil {
		log.Printf("page-err: %v", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, msg)
		return true
	}
	return false
}

// RedirToLoginIfError checks if err is not nil, and if so, redirects the user to the login page
func RedirToLoginIfError(err error, w http.ResponseWriter, r *http.Request) bool {
	if err != nil {
		path := "/login?continue=" + url.PathEscape(r.URL.Path)
		http.Redirect(w, r, path, http.StatusTemporaryRedirect)
		return true
	}
	return false
}

// WriteJSONBytes writes a slice of bytes to the response with json content type.
func WriteJSONBytes(w http.ResponseWriter, b []byte) {
	w.Header().Add("Content-Type", "application/json")
	w.Write(b)
}

// WriteJSONString writes a string to the response with json content type.
func WriteJSONString(w http.ResponseWriter, s string) {
	w.Header().Add("Content-Type", "application/json")
	fmt.Fprint(w, s)
}

func loadAllTemplateFiles() (*template.Template, error) {
	f, err := os.ReadDir(templateDir)
	if err != nil {
		return nil, err
	}
	fnames := make([]string, len(f))
	for i, v := range f {
		fnames[i] = path.Join(templateDir, v.Name())
	}

	fc := template.FuncMap{
		"add": add,
	}

	t, err := template.New("").Funcs(fc).ParseFiles(fnames...)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func add(a, b int) int {
	return a + b
}
