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
