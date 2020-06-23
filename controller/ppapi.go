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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"quizdrum/model"
	"quizdrum/view"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"google.golang.org/protobuf/proto"
)

// SetProfile is the API handler that persists the profile name for this participant
func (c *Controller) SetProfile(w http.ResponseWriter, r *http.Request) {
	u, err := c.P.GetUserFromCookieAndError(r.Cookie("sid"))
	if view.UnauthIfError(err, w, "cookie error, please logout and then login again") {
		return
	}
	r.ParseForm()
	qid, err := strconv.Atoi(r.PostForm["quiz-id"][0])
	if view.Should500(err, w, "could not parse the quiz id") {
		return
	}
	pname := r.FormValue("profile-name")
	if len(pname) == 0 {
		view.Should500(fmt.Errorf("did not get a profile name"), w, "did not get a profile name")
		return
	}
	if view.Should500(c.P.RegisterParticipant(int64(qid), u.GetId(), pname), w, "failed to register the name") {
		return
	}
	fmt.Fprintln(w, "written")
}

// SubmitAnswer submits a response to the question
func (c *Controller) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	u, err := c.P.GetUserFromCookieAndError(r.Cookie("sid"))
	if view.UnauthIfError(err, w, "cookie error, please logout and then login again") {
		return
	}
	r.ParseForm()

	qzid, err := strconv.Atoi(r.FormValue("qz-id"))
	if view.Should500(err, w, "could not parse quiz id") {
		return
	}
	qz, err := c.P.GetQuizWithoutQuestions(int64(qzid))
	if view.Should500(err, w, "could not fetch quiz") {
		return
	}
	if !qz.GetAcceptingResponses() {
		w.WriteHeader(http.StatusConflict)
		fmt.Fprintf(w, "the quiz is not accepting responses right now")
		return
	}

	ans, err := GetAnswerFromPostBody(r.PostForm)
	if view.Should500(err, w, "could not construct the answer from the post body") {
		return
	}

	ans.ResponseTimeS = proto.Int64(time.Now().Unix())
	ans.SolverId = proto.Int64(u.GetId())
	// TODO: validate that the answer type matches the question type
	if ans.GetId() != 0 {
		// Update
		if view.Should500(c.P.UpdateAnswer(ans), w, "could not update the answer") {
			return
		}
		view.WriteJSONString(w, fmt.Sprint(ans.GetId()))
	} else {
		// Create
		aid, err := c.P.CreateAnswer(ans)
		if view.Should500(err, w, "could not store the answer") {
			return
		}
		view.WriteJSONString(w, fmt.Sprint(aid))
	}
}

// GetQuizStatus returns the current question ID and whether answers are being accepted.
func (c *Controller) GetQuizStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	qzid, err := strconv.Atoi(vars["quizid"])
	if view.Should500(err, w, "could not parse quiz id") {
		return
	}
	qz, err := c.P.GetQuizWithoutQuestions(int64(qzid))
	if view.Should500(err, w, "could not fetch quiz") {
		return
	}
	resp := struct {
		QuestionID         int64
		AcceptingResponses bool
	}{
		QuestionID:         qz.GetLiveQuestionId(),
		AcceptingResponses: qz.GetAcceptingResponses(),
	}
	b, err := json.Marshal(resp)
	if view.Should500(err, w, "could not build a json response") {
		return
	}
	view.WriteJSONBytes(w, b)
}

// GetAnswerFromPostBody builds a Answer proto from the submitted form
func GetAnswerFromPostBody(p url.Values) (*model.Answer, error) {
	var ans model.Answer
	// Question ID is mandatory (for creates, but for consistency we are enforcing for updates also)
	qid, err := strconv.Atoi(p["qn-id"][0])
	if err != nil {
		return nil, err
	}
	ans.QuestionId = proto.Int64(int64(qid))

	// Only populate the answer ID if it was supplied as input
	// (it won't be there for creates)
	if val, ok := p["ans-id"]; ok {
		ansid, err := strconv.Atoi(val[0])
		if err == nil {
			ans.Id = proto.Int64(int64(ansid))
		}
		// We ignore the error here, because for creates, the ID comes
		// in as an empty string, which fails to parse.
	}

	if val, ok := p["ans-text"]; ok {
		ans.AnsText = proto.String(val[0])
		ans.Type = model.AnswerType_TEXT_ANSWER.Enum()
	} else if val, ok := p["ans-int64"]; ok {
		ansint, err := strconv.ParseInt(val[0], 10, 64)
		if err != nil {
			return nil, err
		}
		ans.AnsInt = proto.Int64(ansint)
		ans.Type = model.AnswerType_INT64_ANSWER.Enum()
	} else if val, ok := p["ans-float"]; ok {
		ansflt, err := strconv.ParseFloat(val[0], 32)
		if err != nil {
			return nil, err
		}
		ans.AnsFloat = proto.Float32(float32(ansflt))
		ans.Type = model.AnswerType_FLOAT_ANSWER.Enum()
	} else if val, ok := p["ans-bool"]; ok {
		ansbool, err := strconv.ParseBool(val[0])
		if err != nil {
			return nil, err
		}
		ans.AnsBool = proto.Bool(ansbool)
		ans.Type = model.AnswerType_BOOL_ANSWER.Enum()
	} else if val, ok := p["ans-mcq"]; ok {
		ansind, err := strconv.ParseInt(val[0], 10, 64)
		if err != nil {
			return nil, err
		}
		ans.AnsChoiceIndex = proto.Int64(ansind)
		ans.Type = model.AnswerType_MULTIPLE_CHOICE_ANSWER.Enum()
	} else {
		return nil, fmt.Errorf("Did not get any supported answer type")
	}
	return &ans, nil
}
