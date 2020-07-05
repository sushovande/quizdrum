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
	"net/url"
	"quizdrum/model"
	"quizdrum/view"
	"regexp"
	"strconv"

	"github.com/gorilla/mux"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var (
	scoreFormName       = regexp.MustCompile(`ans-([0-9]+)-score`)
	scoreFormNameCustom = regexp.MustCompile(`ans-([0-9]+)-custom-score`)
)

// NewQuiz is the API handler that creates a new Quiz
func (c *Controller) NewQuiz(w http.ResponseWriter, r *http.Request) {
	u, err := c.P.GetUserFromCookieAndError(r.Cookie("sid"))
	if view.UnauthIfError(err, w, "cookie error, please logout and then login again") {
		return
	}
	var qz model.Quiz
	r.ParseForm()
	setQMProfileInQuiz(u, &qz)
	qz.Title = proto.String(r.PostForm["quiz-title"][0])
	qz.HtmlDescription = proto.String(r.PostForm["quiz-descr"][0])
	qzid, err := c.P.CreateQuiz(&qz)
	if view.Should500(err, w, "failed to store") {
		return
	}
	view.WriteJSONString(w, fmt.Sprint(qzid))
}

// NewQuestion is the API handler that creates and persists a new Question
func (c *Controller) NewQuestion(w http.ResponseWriter, r *http.Request) {
	u, err := c.P.GetUserFromCookieAndError(r.Cookie("sid"))
	if view.UnauthIfError(err, w, "cookie error, please logout and then login again") {
		return
	}
	r.ParseForm()
	qn, err := GetQuestionFromPostBody(r.PostForm)
	if view.Should500(err, w, "could not parse the question data") {
		return
	}
	if view.UnauthIfError(c.P.ValidateWritePrivileges(qn.GetQuizId(), u), w, "no write privileges") {
		return
	}

	qnid, err := c.P.CreateQuestion(qn)
	if view.Should500(err, w, "could not save the question") {
		return
	}
	view.WriteJSONString(w, fmt.Sprint(qnid))
}

// GetQuestion is the API handler that reads a question by ID
func (c *Controller) GetQuestion(w http.ResponseWriter, r *http.Request) {
	u, err := c.P.GetUserFromCookieAndError(r.Cookie("sid"))
	if view.UnauthIfError(err, w, "cookie error, please logout and then login again") {
		return
	}
	vars := mux.Vars(r)
	qnid, err := strconv.Atoi(vars["questionid"])
	if view.Should500(err, w, "could not parse the question id") {
		return
	}
	qn, err := c.P.GetQuestionByID(uint(qnid))
	if view.Should500(err, w, "could not find the question") {
		return
	}
	// Note that since this is a QM API, write privileges are still needed.
	if view.UnauthIfError(c.P.ValidateWritePrivileges(qn.GetQuizId(), u), w, "no write privileges") {
		return
	}
	b, err := protojson.Marshal(qn)
	if view.Should500(err, w, "could not format question as json") {
		return
	}
	view.WriteJSONBytes(w, b)
}

// UpdateQuestion is the API handler that persists a modified question
func (c *Controller) UpdateQuestion(w http.ResponseWriter, r *http.Request) {
	u, err := c.P.GetUserFromCookieAndError(r.Cookie("sid"))
	if view.UnauthIfError(err, w, "cookie error, please logout and then login again") {
		return
	}
	r.ParseForm()
	qn, err := GetQuestionFromPostBody(r.PostForm)
	if view.Should500(err, w, "could not parse the question data") {
		return
	}
	qnFromDb, err := c.P.GetQuestionByID(uint(qn.GetId()))
	if view.Should500(err, w, "could not find the question to update") {
		return
	}
	if qn.GetQuizId() != qnFromDb.GetQuizId() {
		view.Should500(fmt.Errorf("Attempt to change quiz id from %v to %v", qnFromDb.GetQuizId(), qn.GetQuizId()),
			w, "you cannot change the quiz id of a question.")
	}
	if view.UnauthIfError(c.P.ValidateWritePrivileges(qn.GetQuizId(), u), w, "no write privileges") {
		return
	}
	if view.Should500(c.P.SaveQuestion(qn), w, "could not save the question") {
		return
	}
	fmt.Fprintln(w, "written")
}

// DeleteQuestion is the API handler that soft-deletes a question
func (c *Controller) DeleteQuestion(w http.ResponseWriter, r *http.Request) {
	u, err := c.P.GetUserFromCookieAndError(r.Cookie("sid"))
	if view.UnauthIfError(err, w, "cookie error, please logout and then login again") {
		return
	}
	vars := mux.Vars(r)
	qnid, err := strconv.Atoi(vars["questionid"])
	if view.Should500(err, w, "could not parse the question id") {
		return
	}
	qn, err := c.P.GetQuestionByID(uint(qnid))
	if view.Should500(err, w, "could not find the question to delete") {
		return
	}
	if view.UnauthIfError(c.P.ValidateWritePrivileges(qn.GetQuizId(), u), w, "no write privileges") {
		return
	}
	if view.Should500(c.P.DeleteQuestion(uint(qnid)), w, "could not delete the question") {
		return
	}
	fmt.Fprintln(w, "deleted")
}

// SetActiveQuestionID sets the active question during a live quiz session.
func (c *Controller) SetActiveQuestionID(w http.ResponseWriter, r *http.Request) {
	u, err := c.P.GetUserFromCookieAndError(r.Cookie("sid"))
	if view.UnauthIfError(err, w, "cookie error, please logout and then login again") {
		return
	}
	vars := mux.Vars(r)
	qzid, err := strconv.Atoi(vars["quizid"])
	if view.Should500(err, w, "could not parse quiz id") {
		return
	}
	qnid, err := strconv.Atoi(vars["questionid"])
	if view.Should500(err, w, "could not parse question id") {
		return
	}
	if view.UnauthIfError(c.P.ValidateWritePrivileges(int64(qzid), u), w, "no write privileges") {
		return
	}
	// TODO: make this atomic if needed
	// TODO: avoid the preloading of questions if inefficient
	qz, err := c.P.GetQuiz(int64(qzid))
	if view.Should500(err, w, "could not load quiz") {
		return
	}
	for _, qn := range qz.GetQuestions() {
		if qn.GetId() == int64(qnid) {
			qz.LiveQuestionId = proto.Int64(int64(qnid))
			qz.AcceptingResponses = proto.Bool(true)
			if view.Should500(c.P.SaveQuiz(qz), w, "could not save quiz") {
				return
			}
			fmt.Fprintln(w, "Saved")
			return
		}
	}

	view.Should500(
		fmt.Errorf("could not find that question %v in the catalog for this quiz %v", qnid, qzid),
		w, "could not find that question id in the catalog for this quiz")
}

// SetAcceptingResponses sets whether the quiz is currently accepting responses or not.
func (c *Controller) SetAcceptingResponses(w http.ResponseWriter, r *http.Request) {
	u, err := c.P.GetUserFromCookieAndError(r.Cookie("sid"))
	if view.UnauthIfError(err, w, "cookie error, please logout and then login again") {
		return
	}
	vars := mux.Vars(r)
	qzid, err := strconv.Atoi(vars["quizid"])
	if view.Should500(err, w, "could not parse quiz id") {
		return
	}
	if view.UnauthIfError(c.P.ValidateWritePrivileges(int64(qzid), u), w, "no write privileges") {
		return
	}
	r.ParseForm()
	ac := false
	if r.FormValue("ar") == "true" {
		ac = true
	}
	qz, err := c.P.GetQuizWithoutQuestions(int64(qzid))
	if view.Should500(err, w, "could not get quiz") {
		return
	}
	qz.AcceptingResponses = proto.Bool(ac)
	if view.Should500(c.P.SaveQuiz(qz), w, "could not save quiz") {
		return
	}
	fmt.Fprintf(w, "set accepting responses to %v", r.FormValue("ar"))
}

// GetAllAnswersForQuestion finds all the answers for this question
func (c *Controller) GetAllAnswersForQuestion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	qnid, err := strconv.Atoi(vars["questionid"])
	if view.Should500(err, w, "could not parse question id") {
		return
	}
	sansa, err := c.P.GetAllAnswersToQuestionID(uint(qnid))
	if view.Should500(err, w, "could not get answers to the question") {
		return
	}

	qz, err := c.P.GetQuizFromQuestionID(uint(qnid))
	if view.Should500(err, w, "could not find the related quiz") {
		return
	}

	type ansDisplay struct {
		AnswerID            int64
		SolverID            int64
		SolverProfileName   string
		AnswerDisplayText   string
		ResponseTimeS       int64
		PointsAwarded       int64
		CustomPointsAwarded bool
	}

	dasp := make([]ansDisplay, 0)
	for _, ans := range sansa {
		var ad ansDisplay
		ad.AnswerID = ans.GetId()
		ad.SolverID = ans.GetSolverId()
		for _, prf := range qz.GetParticipants() {
			if ad.SolverID == int64(prf.GetUserId()) {
				ad.SolverProfileName = prf.GetProfileName()
				break
			}
		}
		ad.AnswerDisplayText = getPrintableStringFromAnswer(ans)
		ad.ResponseTimeS = ans.GetResponseTimeS()
		ad.PointsAwarded = ans.GetPointsAwarded()
		ad.CustomPointsAwarded = !(ans.GetPointsAwarded() == 0 ||
			ans.GetPointsAwarded() == 5 ||
			ans.GetPointsAwarded() == 10)
		dasp = append(dasp, ad)
	}

	c.V.RenderTemplate(w, "qm_answer.html", dasp)
}

// SaveScores stores all the quizmaster awarded points to the answers
func (c *Controller) SaveScores(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	qnid, err := strconv.Atoi(vars["questionid"])
	if view.Should500(err, w, "could not parse question id") {
		return
	}
	sansa, err := c.P.GetAllAnswersToQuestionID(uint(qnid))
	if view.Should500(err, w, "could not get answers to the question") {
		return
	}
	r.ParseForm()
	obtainedScores, err := getIDToScoreMapFromPostForm(r.PostForm)
	if view.Should500(err, w, "could not figure out the score assignment properly") {
		return
	}
	answersToUpdate := make([]*model.Answer, 0)
	for _, ans := range sansa {
		if val, ok := obtainedScores[ans.GetId()]; ok {
			if ans.GetPointsAwarded() != val {
				ans.PointsAwarded = proto.Int64(val)
				answersToUpdate = append(answersToUpdate, ans)
			}
		}
	}
	if view.Should500(c.P.SaveMultipleAnswers(answersToUpdate), w, "could not save the scores") {
		return
	}
	fmt.Fprintln(w, "written")
}

// UpdateQuizProperties stores the quiz properties
func (c *Controller) UpdateQuizProperties(w http.ResponseWriter, r *http.Request) {
	u, err := c.P.GetUserFromCookieAndError(r.Cookie("sid"))
	if view.UnauthIfError(err, w, "cookie error, please logout and then login again") {
		return
	}
	r.ParseForm()
	if len(r.PostForm["qz-title"]) < 1 || len(r.PostForm["qz-descr"]) < 1 ||
		len(r.PostForm["qz-title"][0]) < 1 || len(r.PostForm["qz-descr"][0]) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "quiz title and descr must be specified")
		return
	}
	vars := mux.Vars(r)
	qzid, err := strconv.Atoi(vars["quizid"])
	if view.Should500(err, w, "could not parse quiz id") {
		return
	}
	if view.UnauthIfError(c.P.ValidateWritePrivileges(int64(qzid), u), w, "no write privileges") {
		return
	}
	var qz model.Quiz
	qz.Id = proto.Int64(int64(qzid))
	qz.Title = proto.String(r.PostForm["qz-title"][0])
	qz.HtmlDescription = proto.String(r.PostForm["qz-descr"][0])
	if view.Should500(c.P.SaveQuizMetadata(&qz), w, "could not save the quiz") {
		return
	}
	fmt.Fprint(w, "written")
}

// DeleteQuiz soft-deletes the quiz
func (c *Controller) DeleteQuiz(w http.ResponseWriter, r *http.Request) {
	u, err := c.P.GetUserFromCookieAndError(r.Cookie("sid"))
	if view.UnauthIfError(err, w, "cookie error, please logout and then login again") {
		return
	}
	vars := mux.Vars(r)
	qzid, err := strconv.Atoi(vars["quizid"])
	if view.Should500(err, w, "could not parse quiz id") {
		return
	}
	if view.UnauthIfError(c.P.ValidateWritePrivileges(int64(qzid), u), w, "no write privileges") {
		return
	}
	if view.Should500(c.P.DeleteQuiz(int64(qzid)), w, "could not delete quiz") {
		return
	}
	fmt.Fprint(w, "deleted")
}

// ReinstateQuiz makes the quiz live again
func (c *Controller) ReinstateQuiz(w http.ResponseWriter, r *http.Request) {
	u, err := c.P.GetUserFromCookieAndError(r.Cookie("sid"))
	if view.UnauthIfError(err, w, "cookie error, please logout and then login again") {
		return
	}
	vars := mux.Vars(r)
	qzid, err := strconv.Atoi(vars["quizid"])
	if view.Should500(err, w, "could not parse quiz id") {
		return
	}
	if view.UnauthIfError(c.P.ValidateWritePrivileges(int64(qzid), u), w, "no write privileges") {
		return
	}
	if view.Should500(c.P.ReinstateQuiz(int64(qzid)), w, "could not reinstate quiz") {
		return
	}
	fmt.Fprint(w, "deleted")
}

func getIDToScoreMapFromPostForm(p url.Values) (map[int64]int64, error) {
	resp := make(map[int64]int64)
	for k, v := range p {
		if len(v) < 1 {
			continue
		}
		if matches := scoreFormName.FindStringSubmatch(k); len(matches) > 1 {
			answerID, err := strconv.ParseInt(matches[1], 10, 64)
			if err != nil {
				return nil, err
			}
			if v[0] == "custom" {
				cskey := fmt.Sprintf("ans-%v-custom-score", answerID)
				if _, ok := p[cskey]; !ok {
					return nil, fmt.Errorf("custom score option was chosen but value was not provided")
				}
				csval := p[cskey]
				if len(csval) == 0 || len(csval[0]) == 0 {
					return nil, fmt.Errorf("custom score option was chosen but value was empty")
				}
				score, err := strconv.ParseInt(csval[0], 10, 64)
				if err != nil {
					return nil, err
				}
				resp[answerID] = score
			} else {
				score, err := strconv.ParseInt(v[0], 10, 64)
				if err != nil {
					return nil, err
				}
				resp[answerID] = score
			}
		}
	}
	return resp, nil
}

func getPrintableStringFromAnswer(ans *model.Answer) string {
	switch ans.GetType() {
	case model.AnswerType_TEXT_ANSWER:
		return ans.GetAnsText()
	case model.AnswerType_LONG_TEXT_ANSWER:
		return ans.GetAnsLongtext()
	case model.AnswerType_INT64_ANSWER:
		return strconv.FormatInt(ans.GetAnsInt(), 10)
	case model.AnswerType_FLOAT_ANSWER:
		return strconv.FormatFloat(float64(ans.GetAnsFloat()), 'G', -1, 32)
	case model.AnswerType_BOOL_ANSWER:
		if ans.GetAnsBool() {
			return "True"
		}
		return "False"
	case model.AnswerType_MULTIPLE_CHOICE_ANSWER:
		return fmt.Sprintf("Option %v", (ans.GetAnsChoiceIndex() + 1))
	default:
		return "Error: Invalid Answer"
	}
}

func setQMProfileInQuiz(u *model.User, qz *model.Quiz) {
	var qmf model.QuizmasterProfile
	if u.GetId() != 0 {
		qmf.UserId = proto.Int64(u.GetId())
	}
	if u.GoogleUser.GetSub() != "" {
		qmf.GoogleSub = proto.String(u.GoogleUser.GetSub())
	}
	if u.GoogleUser.GetEmail() != "" {
		qmf.GoogleEmail = proto.String(u.GoogleUser.GetEmail())
	}
	qz.Quizmasters = append(qz.Quizmasters, &qmf)
}
