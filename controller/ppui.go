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
	"quizdrum/model"
	"quizdrum/view"
	"strconv"

	"github.com/gorilla/mux"
	"google.golang.org/protobuf/proto"
)

// RenderCreateProfile is the UI handler that shows the page to allow participant to enter a profile name
// Redirects to the current question page if participant has already chosen a name.
func (c *Controller) RenderCreateProfile(w http.ResponseWriter, r *http.Request) {
	u, err := c.P.GetUserFromCookieAndError(r.Cookie("sid"))
	if view.RedirToLoginIfError(err, w, r) {
		return
	}
	vars := mux.Vars(r)
	qid, err := strconv.Atoi(vars["quizid"])
	if view.Should500(err, w, "could not parse quiz id") {
		return
	}
	q, err := c.P.GetQuiz(int64(qid))
	if view.Should500(err, w, "could not fetch quiz") {
		return
	}

	for _, pp := range q.GetParticipants() {
		if pp.GetUserId() == u.GetId() {
			// already registered
			liveURL := fmt.Sprintf("/participant/quiz/%v/live", qid)
			http.Redirect(w, r, liveURL, http.StatusTemporaryRedirect)
			return
		}
	}

	s := struct {
		U *model.User
		Q *model.Quiz
	}{
		U: u,
		Q: q,
	}

	c.V.RenderTemplate(w, "pp_createprofile.html", s)
}

// RenderLiveQuiz is the UI handler that shows a quiz in progress, with the current
// question selected.
func (c *Controller) RenderLiveQuiz(w http.ResponseWriter, r *http.Request) {
	u, err := c.P.GetUserFromCookieAndError(r.Cookie("sid"))
	if view.RedirToLoginIfError(err, w, r) {
		return
	}
	vars := mux.Vars(r)
	qid, err := strconv.Atoi(vars["quizid"])
	if view.Should500(err, w, "could not parse quiz id") {
		return
	}
	q, err := c.P.GetQuiz(int64(qid))
	if view.Should500(err, w, "could not fetch quiz") {
		return
	}

	profileName := findProfileNameFromQuizAndUser(q, u)
	for _, pp := range q.GetParticipants() {
		if pp.GetUserId() == u.GetId() {
			profileName = pp.GetProfileName()
			break
		}
	}

	var qn *model.Question
	for _, qni := range q.GetQuestions() {
		if qni.GetId() == q.GetLiveQuestionId() {
			qn = qni
			break
		}
	}
	if qn == nil {
		qn = &model.Question{
			Title:    proto.String("The quizmaster has not yet selected a question."),
			HtmlBody: proto.String("Please wait for the quiz to begin."),
			Type:     model.AnswerType_UNKNOWN_ANSWER_TYPE.Enum(),
		}
	}

	var ans *model.Answer
	if qn != nil {
		a, err := c.P.GetAnswerByUserAndQuestion(u, qn)
		// Ignoring errors here, since it could just be the case that the answer does not exist
		if err == nil {
			ans = a
		}
	}

	s := struct {
		U           *model.User
		Q           *model.Quiz
		Qn          *model.Question
		ProfileName string
		Ans         *model.Answer
	}{
		U:           u,
		Q:           q,
		Qn:          qn,
		ProfileName: profileName,
		Ans:         ans,
	}

	c.V.RenderTemplate(w, "pp_live.html", s)
}

// RenderScoreboard shows the scoreboard for the quiz for the participants
func (c *Controller) RenderScoreboard(w http.ResponseWriter, r *http.Request) {
	u, err := c.P.GetUserFromCookieAndError(r.Cookie("sid"))
	if view.RedirToLoginIfError(err, w, r) {
		return
	}
	vars := mux.Vars(r)
	qid, err := strconv.Atoi(vars["quizid"])
	if view.Should500(err, w, "could not parse quiz id") {
		return
	}
	qz, err := c.P.GetQuiz(int64(qid))
	if view.Should500(err, w, "could not fetch quiz") {
		return
	}
	ansmap, err := c.P.GetAllAnswersForSetOfQuestions(qz.GetQuestions())
	if view.Should500(err, w, "could not fetch answers") {
		return
	}

	type participantAndScores struct {
		ParticipantName string
		Total           int64
		Score           []int64
	}
	type scbd struct {
		QuizName      string
		QuestionTitle []string
		PAndScore     []participantAndScores
		U             *model.User
		ProfileName   string
	}

	var board scbd
	board.QuizName = qz.GetTitle()
	board.U = u
	board.ProfileName = findProfileNameFromQuizAndUser(qz, u)

	// First, we arrange the participants in some order
	ppToIndex := make(map[int64]int)
	for i, pp := range qz.GetParticipants() {
		ppToIndex[pp.GetUserId()] = i
		//board.ParticipantName = append(board.ParticipantName, pp.GetProfileName())
	}

	// Next, we arrange the questions in some order
	// TODO: change this to depend on the question_sequence variable
	qnToIndex := make(map[int64]int)
	for i, qn := range qz.GetQuestions() {
		qnToIndex[qn.GetId()] = i
		board.QuestionTitle = append(board.QuestionTitle, qn.GetTitle())
	}

	// Now, we run through the participants and write down their names
	board.PAndScore = make([]participantAndScores, len(ppToIndex))
	for _, pp := range qz.GetParticipants() {
		y := ppToIndex[pp.GetUserId()]
		board.PAndScore[y].ParticipantName = pp.GetProfileName()
		board.PAndScore[y].Score = make([]int64, len(qnToIndex))
	}

	// Now we fill out the score tables
	for _, qn := range qz.GetQuestions() {
		x := qnToIndex[qn.GetId()]
		for _, ans := range ansmap[qn] {
			y := ppToIndex[ans.GetSolverId()]
			board.PAndScore[y].Score[x] = ans.GetPointsAwarded()
			board.PAndScore[y].Total += ans.GetPointsAwarded()
		}
	}

	c.V.RenderTemplate(w, "pp_scoreboard.html", board)
}

func findProfileNameFromQuizAndUser(qz *model.Quiz, u *model.User) string {
	profileName := "unset profile name"
	for _, pp := range qz.GetParticipants() {
		if pp.GetUserId() == u.GetId() {
			profileName = pp.GetProfileName()
			break
		}
	}
	return profileName
}
