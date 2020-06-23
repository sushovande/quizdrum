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
	"net/url"
	"quizdrum/model"
	"strconv"

	"google.golang.org/protobuf/proto"
)

// GetQuestionFromPostBody builds a Question proto from the submitted form
func GetQuestionFromPostBody(p url.Values) (*model.Question, error) {
	var qn model.Question
	// Quiz ID is mandatory (for creates, but for consistency we are enforcing for updates also)
	qid, err := strconv.Atoi(p["quiz-id"][0])
	if err != nil {
		return nil, err
	}

	// If present, get the Question ID too.
	if val, ok := p["qn-id"]; ok && val[0] != "" {
		qnid, err := strconv.Atoi(val[0])
		if err != nil {
			return nil, err
		}
		qn.Id = proto.Int64(int64(qnid))
	}
	qn.Title = proto.String(p["qn-title"][0])
	qn.HtmlBody = proto.String(p["qn-body"][0])
	qn.QuizId = proto.Int64(int64(qid))
	switch p["qn-type"][0] {
	case "text":
		qn.Type = model.AnswerType_TEXT_ANSWER.Enum()
	case "int":
		qn.Type = model.AnswerType_INT64_ANSWER.Enum()
	case "bool":
		qn.Type = model.AnswerType_BOOL_ANSWER.Enum()
	case "mcq":
		qn.Type = model.AnswerType_MULTIPLE_CHOICE_ANSWER.Enum()
		if err = setMcqOptionsFromFormValues(p["mcq-opt"], &qn); err != nil {
			return nil, err
		}
	case "float":
		qn.Type = model.AnswerType_FLOAT_ANSWER.Enum()
	default:
		return nil, fmt.Errorf("unexpected question type: %v", p["qn-type"][0])
	}
	return &qn, nil
}

func setMcqOptionsFromFormValues(f []string, qn *model.Question) error {
	if len(f) == 0 {
		return fmt.Errorf("A multiple choice question must have the options set")
	}

	count := 0
	for _, v := range f {
		if len(v) < 1 {
			continue
		}
		var ch model.AnswerChoice
		ch.HtmlBody = proto.String(v)
		qn.Choices = append(qn.Choices, &ch)
		count++
	}

	if count <= 1 {
		return fmt.Errorf("A multiple choice question must have at least two options")
	}
	return nil
}
