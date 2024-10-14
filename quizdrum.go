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

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"quizdrum/controller"
	"quizdrum/model"
	"quizdrum/view"
	"strings"

	"github.com/gorilla/mux"
)

var port = flag.String("port", "8094", "the port on which the server will listen")
var oauthClientID = flag.String("oauth_client_id", "", "OAuth 2.0 web Client ID obtained from Google Developers site")

func main() {
	flag.Parse()

	if len(*oauthClientID) == 0 {
		b, err := os.ReadFile("oauth_client_id.txt")
		if err != nil {
			panic(fmt.Errorf("--oauth_client_id is empty. You can obtain a client ID from " +
				"https://developers.google.com/identity/sign-in/web/sign-in#create_authorization_credentials" +
				"\nYou may also store the client ID in a file called oauth_client_id.txt in the current directory" +
				" if you don't wish to pass in the command line argument every time."))
		}
		*oauthClientID = strings.TrimSpace(string(b))
	}

	var p model.Persistence
	err := p.Initialize("quizdrum.db", *oauthClientID)
	if err != nil {
		log.Fatal(err)
	}
	defer p.Close()

	var v view.View
	err = v.Initialize()
	if err != nil {
		log.Fatal(err)
	}

	c := controller.Controller{
		P: &p,
		V: &v,
	}

	r := mux.NewRouter()
	r.HandleFunc("/quizmaster/quiz/{quizid}/edit", c.QmEditQuiz)
	r.HandleFunc("/quizmaster/quiz/{quizid}/live", c.QmLive)
	r.HandleFunc("/quizmaster/quiz/{quizid}/scoreboard", c.RenderQMScoreboard)
	r.HandleFunc("/participant/quiz/{quizid}/createprofile", c.RenderCreateProfile)
	r.HandleFunc("/participant/quiz/{quizid}/live", c.RenderLiveQuiz)
	r.HandleFunc("/participant/quiz/{quizid}/scoreboard", c.RenderScoreboard)

	r.HandleFunc("/api/quizmaster/newquiz", c.NewQuiz).Methods("POST")
	r.HandleFunc("/api/quizmaster/quiz/{quizid}/setactive/{questionid}", c.SetActiveQuestionID).Methods("POST")
	r.HandleFunc("/api/quizmaster/quiz/{quizid}/setacceptingresponses", c.SetAcceptingResponses).Methods("POST")
	r.HandleFunc("/api/quizmaster/quiz/{quizid}/updateproperties", c.UpdateQuizProperties).Methods("PUT")
	r.HandleFunc("/api/quizmaster/quiz/{quizid}/delete", c.DeleteQuiz).Methods("DELETE")
	r.HandleFunc("/api/quizmaster/quiz/{quizid}/reinstate", c.ReinstateQuiz).Methods("PUT")
	r.HandleFunc("/api/quizmaster/question/new", c.NewQuestion).Methods("POST")
	r.HandleFunc("/api/quizmaster/question/{questionid}", c.GetQuestion).Methods("GET")
	r.HandleFunc("/api/quizmaster/question/{questionid}/delete", c.DeleteQuestion).Methods("DELETE")
	r.HandleFunc("/api/quizmaster/question/{questionid}/update", c.UpdateQuestion).Methods("PUT")
	r.HandleFunc("/api/quizmaster/question/{questionid}/getallanswers", c.GetAllAnswersForQuestion).Methods("GET")
	r.HandleFunc("/api/quizmaster/question/{questionid}/savescores", c.SaveScores).Methods("POST")

	r.HandleFunc("/api/participant/set-profile", c.SetProfile).Methods("POST")
	r.HandleFunc("/api/participant/submit-answer", c.SubmitAnswer).Methods("POST")
	r.HandleFunc("/api/participant/quiz/{quizid}/getstatus", c.GetQuizStatus).Methods("GET")
	r.HandleFunc("/api/common/guest-login", c.HandleGuestLogin).Methods("POST")
	r.HandleFunc("/api/common/oauth-login", c.HandleOauthLogin).Methods("POST")

	r.HandleFunc("/", c.RenderHomepage)
	r.HandleFunc("/login", c.RenderLogin)
	r.HandleFunc("/logout", c.HandleLogout)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fmt.Println("Server ready.")
	log.Fatal(http.ListenAndServe(":"+*port, r))
}
