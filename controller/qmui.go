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
	"html/template"
	"net/http"
	"quizdrum/model"
	"quizdrum/view"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"google.golang.org/protobuf/proto"
)

// QmEditQuiz is the handler that renders the UI to edit a quiz
func (c *Controller) QmEditQuiz(w http.ResponseWriter, r *http.Request) {
	u, err := c.P.GetUserFromCookieAndError(r.Cookie("sid"))
	if view.RedirToLoginIfError(err, w, r) {
		return
	}
	vars := mux.Vars(r)
	qzid, err := strconv.Atoi(vars["quizid"])
	if view.Should500(err, w, "could not parse quiz id") {
		return
	}
	if view.UnauthIfError(c.P.ValidateWritePrivileges(int64(qzid), u), w,
		"You do not have access to edit this quiz. Please <a href='/logout'>Logout</a>"+
			" and then log in again with an account that has access.") {
		return
	}
	q, err := c.P.GetQuiz(int64(qzid))
	if view.Should500(err, w, "could not fetch quiz") {
		return
	}
	s := struct {
		U *model.User
		Q *model.Quiz
	}{
		U: u,
		Q: q,
	}

	c.V.RenderTemplate(w, "qm_editquiz.html", s)
}

// QmLive renders the UI for the quizmaster to control a live quiz
func (c *Controller) QmLive(w http.ResponseWriter, r *http.Request) {
	u, err := c.P.GetUserFromCookieAndError(r.Cookie("sid"))
	if view.RedirToLoginIfError(err, w, r) {
		return
	}
	vars := mux.Vars(r)
	qzid, err := strconv.Atoi(vars["quizid"])
	if view.Should500(err, w, "could not parse quiz id") {
		return
	}
	if view.UnauthIfError(c.P.ValidateWritePrivileges(int64(qzid), u), w,
		"You do not have access to present this quiz. Please <a href='/logout'>Logout</a>"+
			" and then log in again with an account that has access.") {
		return
	}
	q, err := c.P.GetQuiz(int64(qzid))
	if view.Should500(err, w, "could not fetch quiz") {
		return
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
			Id:       proto.Int64(-1),
			Title:    proto.String("To begin, click the Next Question button."),
			HtmlBody: proto.String("The question will be revealed to the participants and they will be able to respond."),
			Type:     model.AnswerType_UNKNOWN_ANSWER_TYPE.Enum(),
		}
	}

	s := struct {
		U           *model.User
		Q           *model.Quiz
		Qn          *model.Question
		QuestionIds template.JS
	}{
		U:           u,
		Q:           q,
		Qn:          qn,
		QuestionIds: template.JS(getQuestionSequence(q)),
	}

	c.V.RenderTemplate(w, "qm_live.html", s)
}

func getQuestionSequence(q *model.Quiz) string {
	qnids := make([]string, 0)
	for _, qn := range q.GetQuestions() {
		qnids = append(qnids, strconv.FormatInt(qn.GetId(), 10))
	}
	return "var allQuestionIds = [ " + strings.Join(qnids, ",") + "]"
}

// RenderQMScoreboard shows the scoreboard for the quiz for the quizmaster
func (c *Controller) RenderQMScoreboard(w http.ResponseWriter, r *http.Request) {
	u, err := c.P.GetUserFromCookieAndError(r.Cookie("sid"))
	if view.RedirToLoginIfError(err, w, r) {
		return
	}
	vars := mux.Vars(r)
	qzid, err := strconv.Atoi(vars["quizid"])
	if view.Should500(err, w, "could not parse quiz id") {
		return
	}
	qz, err := c.P.GetQuiz(int64(qzid))
	if view.Should500(err, w, "could not fetch quiz") {
		return
	}
	if view.UnauthIfError(c.P.ValidateWritePrivileges(int64(qzid), u), w,
		"You do not have access to present this quiz. Please <a href='/logout'>Logout</a>"+
			" and then log in again with an account that has access.") {
		return
	}
	ansmap, err := c.P.GetAllAnswersForSetOfQuestions(qz.GetQuestions())
	if view.Should500(err, w, "could not fetch answers") {
		return
	}

	// TODO: refactor this copy-pasted code from ppui.go
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
	}

	var board scbd
	board.QuizName = qz.GetTitle()
	board.U = u

	// First, we arrange the participants in some order
	ppToIndex := make(map[int64]int)
	for i, pp := range qz.GetParticipants() {
		ppToIndex[pp.GetUserId()] = i
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

	c.V.RenderTemplate(w, "qm_scoreboard.html", board)
}
