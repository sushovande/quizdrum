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
	"net/http/httptest"
	"quizdrum/model"
	"quizdrum/view"
	"strconv"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"google.golang.org/protobuf/proto"
)

type savedHTTPResponse struct {
	statuscode int
	resptext   string
	cookie     *http.Cookie
}

// callController is a helper method that calls the given controller function
// with the http verb and parameters.
func callController(verb string, path string, body string, cookie *http.Cookie,
	vars map[string]string, f func(http.ResponseWriter, *http.Request)) savedHTTPResponse {
	var r savedHTTPResponse
	a := strings.NewReader(body)
	req := httptest.NewRequest(verb, path, a)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req = mux.SetURLVars(req, vars)
	resp := httptest.NewRecorder()
	f(resp, req)
	r.statuscode = resp.Result().StatusCode
	r.resptext = resp.Body.String()
	for _, ck := range resp.Result().Cookies() {
		if ck.Name == "sid" {
			r.cookie = ck
		}
	}

	return r
}

func TestUserJourney(t *testing.T) {
	var p model.Persistence
	p.Initialize(":memory:", "oauth_client_fake_id")
	var v view.View
	v.Initialize()
	c := Controller{
		P: &p, V: &v,
	}

	var loginCookie *http.Cookie
	// First, the QM logs in
	{
		r := callController("POST", "/api/common/guest-login", "", nil, nil, c.HandleGuestLogin)
		if r.resptext != "1" {
			t.Fatalf("unexpected user id response. want %v, got %v", "1", r.resptext)
		}
		if r.cookie == nil || r.cookie.Value == "" {
			t.Fatalf("User logged in, but cookie is empty")
		}
		loginCookie = r.cookie
	}

	// Then, the QM creates a quiz
	var qzid int64
	{
		r := callController("POST", "/api/quizmaster/newquiz",
			"quiz-title=GreatQuiz&quiz-descr=AmazingQuizIsHere", loginCookie, nil, c.NewQuiz)
		if r.statuscode != 200 {
			t.Fatalf("Failed to create quiz. HTTP %v. %v", r.statuscode, r.resptext)
		}
		if len(r.resptext) == 0 {
			t.Fatalf("Got empty response for create quiz.")
		}
		var err error
		qzid, err = strconv.ParseInt(r.resptext, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Add a question
	var qnid int64
	{
		r := callController("POST", "/api/quizmaster/question/new",
			fmt.Sprintf("quiz-id=%v&qn-title=Quest&qn-body=WhoIsThePresident&qn-type=text", qzid),
			loginCookie, nil, c.NewQuestion)
		if r.statuscode != 200 {
			t.Fatalf("Failed to create question. HTTP %v. %v", r.statuscode, r.resptext)
		}
		if len(r.resptext) == 0 {
			t.Fatalf("Got empty response for create question.")
		}
		var err error
		qnid, err = strconv.ParseInt(r.resptext, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Set question as active
	{
		vars := map[string]string{"quizid": fmt.Sprint(qzid), "questionid": fmt.Sprint(qnid)}
		r := callController("POST",
			fmt.Sprintf("/api/quizmaster/quiz/%v/setactive/%v", qzid, qnid),
			"", loginCookie, vars, c.SetActiveQuestionID)
		if r.statuscode != 200 {
			t.Fatalf("Failed to create question. HTTP %v. %v", r.statuscode, r.resptext)
		}
	}

	// A participant logs in
	var ppCookie *http.Cookie
	{
		r := callController("POST", "/api/common/guest-login", "", nil, nil, c.HandleGuestLogin)
		ppCookie = r.cookie
	}

	// The participant registers for the quiz
	{
		r := callController("POST", "/api/participant/set-profile",
			fmt.Sprintf("quiz-id=%v&profile-name=Party", qzid), ppCookie, nil, c.SetProfile)
		if r.statuscode != 200 {
			t.Fatalf("Failed to create profile. HTTP %v. %v", r.statuscode, r.resptext)
		}
	}

	// The participant answers the question
	var ansid int64
	{
		r := callController("POST", "/api/participant/submit-answer",
			fmt.Sprintf("qz-id=%v&qn-id=%v&ans-text=MrPrez", qzid, qnid), ppCookie, nil, c.SubmitAnswer)
		if r.statuscode != 200 {
			t.Fatalf("Failed to submit ans. HTTP %v. %v", r.statuscode, r.resptext)
		}
		var err error
		ansid, err = strconv.ParseInt(r.resptext, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
	}

	// TODO: Figure out how to test GET calls, since the template files may not be found.
	// QM stops accepting responses
	{
		vars := map[string]string{"quizid": fmt.Sprint(qzid)}
		r := callController("POST",
			fmt.Sprintf("/api/quizmaster/quiz/%v/setacceptingresponses", qzid),
			fmt.Sprintf("ar=false"),
			loginCookie, vars, c.SetAcceptingResponses)
		if r.statuscode != 200 {
			t.Fatalf("Failed to set accepting responses. HTTP %v. %v", r.statuscode, r.resptext)
		}
	}

	// The participant tries to update the answer but fails
	{
		r := callController("POST", "/api/participant/submit-answer",
			fmt.Sprintf("qz-id=%v&qn-id=%v&ans-id=%v&ans-text=MrFriend", qzid, qnid, ansid),
			ppCookie, nil, c.SubmitAnswer)
		if r.statuscode != 409 {
			t.Fatalf("want: HTTP 409. got: HTTP %v. %v", r.statuscode, r.resptext)
		}
	}

	// The QM grades the answer
	{
		vars := map[string]string{"questionid": fmt.Sprint(qnid)}
		r := callController("POST",
			fmt.Sprintf("/api/quizmaster/question/%v/savescores", qnid),
			fmt.Sprintf("ans-%v-score=5", ansid),
			loginCookie, vars, c.SaveScores)
		if r.statuscode != 200 {
			t.Fatalf("Failed to save scores. HTTP %v. %v", r.statuscode, r.resptext)
		}
	}
}

func TestAutomaticGradingJourney(t *testing.T) {
	var p model.Persistence
	p.Initialize(":memory:", "oauth_client_fake_id")
	var v view.View
	v.Initialize()
	c := Controller{
		P: &p, V: &v,
	}

	var loginCookie *http.Cookie
	// 1. QM logs in
	{
		r := callController("POST", "/api/common/guest-login", "", nil, nil, c.HandleGuestLogin)
		if r.statuscode != 200 || r.cookie == nil {
			t.Fatalf("QM login failed")
		}
		loginCookie = r.cookie
	}

	// 2. QM creates quiz
	var qzid int64
	{
		r := callController("POST", "/api/quizmaster/newquiz",
			"quiz-title=AutoGradingQuiz&quiz-descr=TestingAutoGrading", loginCookie, nil, c.NewQuiz)
		if r.statuscode != 200 {
			t.Fatalf("Failed to create quiz: %v", r.resptext)
		}
		var err error
		qzid, err = strconv.ParseInt(r.resptext, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
	}

	// 3. QM creates Question 1: True/False with True=10, False=0
	var qn1Id int64
	{
		body := fmt.Sprintf("quiz-id=%v&qn-title=Q1Bool&qn-body=TheSkyIsBlue&qn-type=bool&bool-true-pts=10&bool-false-pts=0", qzid)
		r := callController("POST", "/api/quizmaster/question/new", body, loginCookie, nil, c.NewQuestion)
		if r.statuscode != 200 {
			t.Fatalf("Failed to create Q1: %v", r.resptext)
		}
		var err error
		qn1Id, err = strconv.ParseInt(r.resptext, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Check Q1 stored points in DB
	qn1, err := p.GetQuestionByID(uint(qn1Id))
	if err != nil {
		t.Fatal(err)
	}
	if qn1.GetTruePoints() != 10 || qn1.GetFalsePoints() != 0 {
		t.Fatalf("Q1 points wrong: true=%v, false=%v", qn1.GetTruePoints(), qn1.GetFalsePoints())
	}

	// 4. QM creates Question 2: MCQ with Option 1=0, Option 2=5, Option 3=20
	var qn2Id int64
	{
		body := fmt.Sprintf("quiz-id=%v&qn-title=Q2MCQ&qn-body=PickOption&qn-type=mcq&mcq-opt=Red&mcq-pts=0&mcq-opt=Green&mcq-pts=5&mcq-opt=Blue&mcq-pts=20", qzid)
		r := callController("POST", "/api/quizmaster/question/new", body, loginCookie, nil, c.NewQuestion)
		if r.statuscode != 200 {
			t.Fatalf("Failed to create Q2: %v", r.resptext)
		}
		qn2Id, err = strconv.ParseInt(r.resptext, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Check Q2 stored points in DB
	qn2, err := p.GetQuestionByID(uint(qn2Id))
	if err != nil {
		t.Fatal(err)
	}
	if len(qn2.GetChoices()) != 3 {
		t.Fatalf("Q2 choices count want 3, got %v", len(qn2.GetChoices()))
	}
	if qn2.GetChoices()[0].GetPoints() != 0 || qn2.GetChoices()[1].GetPoints() != 5 || qn2.GetChoices()[2].GetPoints() != 20 {
		t.Fatalf("Q2 choice points unexpected: %v, %v, %v",
			qn2.GetChoices()[0].GetPoints(), qn2.GetChoices()[1].GetPoints(), qn2.GetChoices()[2].GetPoints())
	}

	// 5. QM creates Question 3: MCQ with points defaulting to 0
	var qn3Id int64
	{
		body := fmt.Sprintf("quiz-id=%v&qn-title=Q3MCQDefault&qn-body=PickDefault&qn-type=mcq&mcq-opt=First&mcq-opt=Second", qzid)
		r := callController("POST", "/api/quizmaster/question/new", body, loginCookie, nil, c.NewQuestion)
		if r.statuscode != 200 {
			t.Fatalf("Failed to create Q3: %v", r.resptext)
		}
		qn3Id, err = strconv.ParseInt(r.resptext, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
	}
	qn3, err := p.GetQuestionByID(uint(qn3Id))
	if err != nil {
		t.Fatal(err)
	}
	if qn3.GetChoices()[0].GetPoints() != 0 || qn3.GetChoices()[1].GetPoints() != 0 {
		t.Fatalf("Q3 choice points should default to 0: %v, %v",
			qn3.GetChoices()[0].GetPoints(), qn3.GetChoices()[1].GetPoints())
	}

	// 6. Participant logs in and registers
	var ppCookie *http.Cookie
	{
		r := callController("POST", "/api/common/guest-login", "", nil, nil, c.HandleGuestLogin)
		ppCookie = r.cookie
		r = callController("POST", "/api/participant/set-profile",
			fmt.Sprintf("quiz-id=%v&profile-name=Alice", qzid), ppCookie, nil, c.SetProfile)
		if r.statuscode != 200 {
			t.Fatalf("Failed to register participant: %v", r.resptext)
		}
	}

	// 7. QM activates Q1 (True/False)
	{
		vars := map[string]string{"quizid": fmt.Sprint(qzid), "questionid": fmt.Sprint(qn1Id)}
		r := callController("POST", fmt.Sprintf("/api/quizmaster/quiz/%v/setactive/%v", qzid, qn1Id),
			"", loginCookie, vars, c.SetActiveQuestionID)
		if r.statuscode != 200 {
			t.Fatalf("Failed to activate Q1: %v", r.resptext)
		}
	}

	// 8. Participant submits True for Q1
	var ans1Id int64
	{
		body := fmt.Sprintf("qz-id=%v&qn-id=%v&ans-bool=true", qzid, qn1Id)
		r := callController("POST", "/api/participant/submit-answer", body, ppCookie, nil, c.SubmitAnswer)
		if r.statuscode != 200 {
			t.Fatalf("Failed to submit Q1 answer: %v", r.resptext)
		}
		ans1Id, err = strconv.ParseInt(r.resptext, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Requirement 3: Participant checks scorecard while question in progress -> should NOT see the 10 points!
	{
		vars := map[string]string{"quizid": fmt.Sprint(qzid)}
		r := callController("GET", fmt.Sprintf("/participant/quiz/%v/scoreboard", qzid), "", ppCookie, vars, c.RenderScoreboard)
		if r.statuscode != 200 {
			t.Fatalf("Failed to render scoreboard: %v", r.resptext)
		}
		// The scoreboard table should NOT have 10 in total or score column for Alice
		// In fact, total should be 0!
		if strings.Contains(r.resptext, `<td class="mdc-data-table__cell">10</td>`) {
			t.Fatalf("Scorecard should NOT show 10 points before QM saves scores or next question!")
		}
		if !strings.Contains(r.resptext, `<td class="mdc-data-table__cell">0</td>`) {
			t.Fatalf("Scorecard should show 0 points initially")
		}
	}

	// Requirement 2: QM loads answers for Q1 -> response card has 10 points automatically populated
	{
		vars := map[string]string{"questionid": fmt.Sprint(qn1Id)}
		r := callController("GET", fmt.Sprintf("/api/quizmaster/question/%v/getallanswers", qn1Id), "", loginCookie, vars, c.GetAllAnswersForQuestion)
		if r.statuscode != 200 {
			t.Fatalf("Failed to get all answers for Q1: %v", r.resptext)
		}
		// Check that ans-<ans1Id>-score-10 has checked attribute
		if !strings.Contains(r.resptext, fmt.Sprintf(`id="ans-%v-score-10"`, ans1Id)) {
			t.Fatalf("Expected radio 10 in QM view")
		}
		// Confirm radio 10 has checked:
		radio10Part := fmt.Sprintf(`id="ans-%v-score-10"`, ans1Id)
		idx := strings.Index(r.resptext, radio10Part)
		snippet := r.resptext[idx : idx+100]
		if !strings.Contains(snippet, "checked") {
			t.Fatalf("Expected radio 10 to be checked in snippet: %v", snippet)
		}
	}

	// QM hits "Save Scores" for Q1
	{
		vars := map[string]string{"questionid": fmt.Sprint(qn1Id)}
		r := callController("POST", fmt.Sprintf("/api/quizmaster/question/%v/savescores", qn1Id),
			fmt.Sprintf("ans-%v-score=10", ans1Id), loginCookie, vars, c.SaveScores)
		if r.statuscode != 200 {
			t.Fatalf("Failed to save scores for Q1: %v", r.resptext)
		}
	}

	// Now participant checks scoreboard -> should see 10 points!
	{
		vars := map[string]string{"quizid": fmt.Sprint(qzid)}
		r := callController("GET", fmt.Sprintf("/participant/quiz/%v/scoreboard", qzid), "", ppCookie, vars, c.RenderScoreboard)
		if r.statuscode != 200 {
			t.Fatalf("Failed to render scoreboard: %v", r.resptext)
		}
		if !strings.Contains(r.resptext, `<td class="mdc-data-table__cell">10</td>`) {
			t.Fatalf("Scorecard should show 10 points after QM saved scores!")
		}
	}

	// 9. QM activates Q2 (MCQ)
	{
		vars := map[string]string{"quizid": fmt.Sprint(qzid), "questionid": fmt.Sprint(qn2Id)}
		r := callController("POST", fmt.Sprintf("/api/quizmaster/quiz/%v/setactive/%v", qzid, qn2Id),
			"", loginCookie, vars, c.SetActiveQuestionID)
		if r.statuscode != 200 {
			t.Fatalf("Failed to activate Q2: %v", r.resptext)
		}
	}

	// Participant answers Option 2 (Blue, index 2, allocated 20 points)
	var ans2Id int64
	{
		body := fmt.Sprintf("qz-id=%v&qn-id=%v&ans-mcq=2", qzid, qn2Id)
		r := callController("POST", "/api/participant/submit-answer", body, ppCookie, nil, c.SubmitAnswer)
		if r.statuscode != 200 {
			t.Fatalf("Failed to submit Q2 answer: %v", r.resptext)
		}
		ans2Id, err = strconv.ParseInt(r.resptext, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Check scoreboard before QM saves -> Q2 score should not be shown (Total still 10)
	{
		vars := map[string]string{"quizid": fmt.Sprint(qzid)}
		r := callController("GET", fmt.Sprintf("/participant/quiz/%v/scoreboard", qzid), "", ppCookie, vars, c.RenderScoreboard)
		if strings.Contains(r.resptext, `<td class="mdc-data-table__cell">30</td>`) {
			t.Fatalf("Scorecard should NOT include auto-points of 20 before saving!")
		}
	}

	// Requirement 2: QM views Q2 answers -> 20 points is auto-populated as custom score!
	{
		vars := map[string]string{"questionid": fmt.Sprint(qn2Id)}
		r := callController("GET", fmt.Sprintf("/api/quizmaster/question/%v/getallanswers", qn2Id), "", loginCookie, vars, c.GetAllAnswersForQuestion)
		if r.statuscode != 200 {
			t.Fatalf("Failed to get all answers for Q2: %v", r.resptext)
		}
		// Custom radio should be checked and value should be 20
		customRadioPart := fmt.Sprintf(`id="ans-%v-score-custom"`, ans2Id)
		idx := strings.Index(r.resptext, customRadioPart)
		if idx == -1 {
			t.Fatalf("Custom radio not found for ans2")
		}
		snippet := r.resptext[idx : idx+150]
		if !strings.Contains(snippet, "checked") {
			t.Fatalf("Expected custom radio to be checked in snippet: %v", snippet)
		}
		customInputPart := fmt.Sprintf(`id="ans-%v-custom-score"`, ans2Id)
		idxInput := strings.Index(r.resptext, customInputPart)
		if idxInput == -1 {
			t.Fatalf("Custom input not found for ans2")
		}
		snippetInput := r.resptext[idxInput : idxInput+150]
		if !strings.Contains(snippetInput, `value="20"`) {
			t.Fatalf("Expected custom input to have value=\"20\" in snippet: %v", snippetInput)
		}
	}

	// Requirement 2 (override): QM overrides score to custom 7!
	{
		vars := map[string]string{"questionid": fmt.Sprint(qn2Id)}
		body := fmt.Sprintf("ans-%v-score=custom&ans-%v-custom-score=7", ans2Id, ans2Id)
		r := callController("POST", fmt.Sprintf("/api/quizmaster/question/%v/savescores", qn2Id),
			body, loginCookie, vars, c.SaveScores)
		if r.statuscode != 200 {
			t.Fatalf("Failed to save overridden scores for Q2: %v", r.resptext)
		}
	}

	// Check scoreboard -> Total should now be 17 (10 from Q1 + 7 from Q2)
	{
		vars := map[string]string{"quizid": fmt.Sprint(qzid)}
		r := callController("GET", fmt.Sprintf("/participant/quiz/%v/scoreboard", qzid), "", ppCookie, vars, c.RenderScoreboard)
		if !strings.Contains(r.resptext, `<td class="mdc-data-table__cell">17</td>`) {
			t.Fatalf("Scorecard should show 17 points after QM saved overridden score! Got:\n%v", r.resptext)
		}
	}

	// 10. Requirement 3 test with Next Question:
	// Update Q3 to give 8 points to choice 1
	qn3.Choices[1].Points = proto.Int64(8)
	p.SaveQuestion(qn3)

	// Activate Q3
	{
		vars := map[string]string{"quizid": fmt.Sprint(qzid), "questionid": fmt.Sprint(qn3Id)}
		r := callController("POST", fmt.Sprintf("/api/quizmaster/quiz/%v/setactive/%v", qzid, qn3Id),
			"", loginCookie, vars, c.SetActiveQuestionID)
		if r.statuscode != 200 {
			t.Fatalf("Failed to activate Q3: %v", r.resptext)
		}
	}

	// Participant answers choice 1 (8 points) for Q3
	{
		body := fmt.Sprintf("qz-id=%v&qn-id=%v&ans-mcq=1", qzid, qn3Id)
		r := callController("POST", "/api/participant/submit-answer", body, ppCookie, nil, c.SubmitAnswer)
		if r.statuscode != 200 {
			t.Fatalf("Failed to submit Q3 answer: %v", r.resptext)
		}
	}

	// Before Next Question is hit: scoreboard must NOT show the 8 points (total is still 17)
	{
		vars := map[string]string{"quizid": fmt.Sprint(qzid)}
		r := callController("GET", fmt.Sprintf("/participant/quiz/%v/scoreboard", qzid), "", ppCookie, vars, c.RenderScoreboard)
		if strings.Contains(r.resptext, `<td class="mdc-data-table__cell">25</td>`) {
			t.Fatalf("Scorecard should NOT show 25 points before Next Question or Save Scores!")
		}
		if !strings.Contains(r.resptext, `<td class="mdc-data-table__cell">17</td>`) {
			t.Fatalf("Scorecard should still show 17 points")
		}
	}

	// QM hits Next Question (which activates Q1 again)
	{
		vars := map[string]string{"quizid": fmt.Sprint(qzid), "questionid": fmt.Sprint(qn1Id)}
		r := callController("POST", fmt.Sprintf("/api/quizmaster/quiz/%v/setactive/%v", qzid, qn1Id),
			"", loginCookie, vars, c.SetActiveQuestionID)
		if r.statuscode != 200 {
			t.Fatalf("Failed to set active question: %v", r.resptext)
		}
	}

	// After Next Question is hit: scoreboard must now show the 8 points (total = 17 + 8 = 25)!
	{
		vars := map[string]string{"quizid": fmt.Sprint(qzid)}
		r := callController("GET", fmt.Sprintf("/participant/quiz/%v/scoreboard", qzid), "", ppCookie, vars, c.RenderScoreboard)
		if !strings.Contains(r.resptext, `<td class="mdc-data-table__cell">25</td>`) {
			t.Fatalf("Scorecard should now show 25 points after Next Question! Got:\n%v", r.resptext)
		}
	}

	// 11. QM comes back to already-finalized Q1 to change the score from 10 to 3
	{
		// QM views Q1 answers again -> shows saved score of 10
		vars := map[string]string{"questionid": fmt.Sprint(qn1Id)}
		r := callController("GET", fmt.Sprintf("/api/quizmaster/question/%v/getallanswers", qn1Id), "", loginCookie, vars, c.GetAllAnswersForQuestion)
		if r.statuscode != 200 {
			t.Fatalf("Failed to get all answers for Q1: %v", r.resptext)
		}
		if !strings.Contains(r.resptext, fmt.Sprintf(`id="ans-%v-score-10"`, ans1Id)) {
			t.Fatalf("Expected radio 10 in QM view for Q1")
		}

		// QM overrides score to custom 3 and saves again
		body := fmt.Sprintf("ans-%v-score=custom&ans-%v-custom-score=3", ans1Id, ans1Id)
		r = callController("POST", fmt.Sprintf("/api/quizmaster/question/%v/savescores", qn1Id),
			body, loginCookie, vars, c.SaveScores)
		if r.statuscode != 200 {
			t.Fatalf("Failed to re-save scores for Q1: %v", r.resptext)
		}
	}

	// Scoreboard now reflects the re-saved score (total was 25, Q1 changed from 10 to 3, so total is now 18)
	{
		vars := map[string]string{"quizid": fmt.Sprint(qzid)}
		r := callController("GET", fmt.Sprintf("/participant/quiz/%v/scoreboard", qzid), "", ppCookie, vars, c.RenderScoreboard)
		if !strings.Contains(r.resptext, `<td class="mdc-data-table__cell">18</td>`) {
			t.Fatalf("Scorecard should reflect re-saved score (total 18). Got:\n%v", r.resptext)
		}
	}
}
