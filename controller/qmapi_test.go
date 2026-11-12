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
	"quizdrum/model"
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestGetPrintableStringFromAnswer(t *testing.T) {
	testCases := []struct {
		ans  *model.Answer
		want string
	}{
		{
			ans: &model.Answer{
				Type:    model.AnswerType_TEXT_ANSWER.Enum(),
				AnsText: proto.String("foofoo"),
			},
			want: "foofoo",
		},
		{
			ans: &model.Answer{
				Type:   model.AnswerType_INT64_ANSWER.Enum(),
				AnsInt: proto.Int64(234),
			},
			want: "234",
		},
		{
			ans: &model.Answer{
				Type:   model.AnswerType_INT64_ANSWER.Enum(),
				AnsInt: proto.Int64(23456789012345678),
			},
			want: "23456789012345678",
		},
		{
			ans: &model.Answer{
				Type:    model.AnswerType_BOOL_ANSWER.Enum(),
				AnsBool: proto.Bool(true),
			},
			want: "True",
		},
		{
			ans: &model.Answer{
				Type:           model.AnswerType_MULTIPLE_CHOICE_ANSWER.Enum(),
				AnsChoiceIndex: proto.Int64(2),
			},
			want: "Option 3",
		},
		{
			ans: &model.Answer{
				Type:        model.AnswerType_LONG_TEXT_ANSWER.Enum(),
				AnsLongtext: proto.String("actually small"),
			},
			want: "actually small",
		},
		{
			ans: &model.Answer{
				Type:     model.AnswerType_FLOAT_ANSWER.Enum(),
				AnsFloat: proto.Float32(2.3456),
			},
			want: "2.3456",
		},
		{
			ans: &model.Answer{
				Type:     model.AnswerType_FLOAT_ANSWER.Enum(),
				AnsFloat: proto.Float32(-7.891E35),
			},
			want: "-7.891E+35",
		},
		{
			ans: &model.Answer{
				Type:     model.AnswerType_FLOAT_ANSWER.Enum(),
				AnsFloat: proto.Float32(-0.005),
			},
			want: "-0.005",
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%v %v", tc.ans.GetType(), tc.want), func(t *testing.T) {
			if got := getPrintableStringFromAnswer(tc.ans); got != tc.want {
				t.Errorf("got: %v; want %v", got, tc.want)
			}
		})
	}
}

func TestGetAutomaticPointsForAnswer(t *testing.T) {
	// MCQ Question
	mcqQn := &model.Question{
		Type: model.AnswerType_MULTIPLE_CHOICE_ANSWER.Enum(),
		Choices: []*model.AnswerChoice{
			{HtmlBody: proto.String("A"), Points: proto.Int64(0)},
			{HtmlBody: proto.String("B"), Points: proto.Int64(10)},
			{HtmlBody: proto.String("C"), Points: proto.Int64(-5)},
		},
	}

	// MCQ Answer Option 0 (0 pts) -> hasAuto = false (since points are zero)
	ans0 := &model.Answer{
		Type:           model.AnswerType_MULTIPLE_CHOICE_ANSWER.Enum(),
		AnsChoiceIndex: proto.Int64(0),
	}
	pts, hasAuto := GetAutomaticPointsForAnswer(mcqQn, ans0)
	if hasAuto || pts != 0 {
		t.Errorf("Option 0 want pts=0, hasAuto=false; got pts=%d, hasAuto=%v", pts, hasAuto)
	}

	// MCQ Answer Option 1 (10 pts) -> hasAuto = true
	ans1 := &model.Answer{
		Type:           model.AnswerType_MULTIPLE_CHOICE_ANSWER.Enum(),
		AnsChoiceIndex: proto.Int64(1),
	}
	pts, hasAuto = GetAutomaticPointsForAnswer(mcqQn, ans1)
	if !hasAuto || pts != 10 {
		t.Errorf("Option 1 want pts=10, hasAuto=true; got pts=%d, hasAuto=%v", pts, hasAuto)
	}

	// MCQ Answer Option 2 (-5 pts) -> hasAuto = true
	ans2 := &model.Answer{
		Type:           model.AnswerType_MULTIPLE_CHOICE_ANSWER.Enum(),
		AnsChoiceIndex: proto.Int64(2),
	}
	pts, hasAuto = GetAutomaticPointsForAnswer(mcqQn, ans2)
	if !hasAuto || pts != -5 {
		t.Errorf("Option 2 want pts=-5, hasAuto=true; got pts=%d, hasAuto=%v", pts, hasAuto)
	}

	// True/False Question
	boolQn := &model.Question{
		Type:        model.AnswerType_BOOL_ANSWER.Enum(),
		TruePoints:  proto.Int64(15),
		FalsePoints: proto.Int64(0),
	}

	// True answer -> 15 pts, hasAuto = true
	ansTrue := &model.Answer{
		Type:    model.AnswerType_BOOL_ANSWER.Enum(),
		AnsBool: proto.Bool(true),
	}
	pts, hasAuto = GetAutomaticPointsForAnswer(boolQn, ansTrue)
	if !hasAuto || pts != 15 {
		t.Errorf("True answer want pts=15, hasAuto=true; got pts=%d, hasAuto=%v", pts, hasAuto)
	}

	// False answer -> 0 pts, hasAuto = false
	ansFalse := &model.Answer{
		Type:    model.AnswerType_BOOL_ANSWER.Enum(),
		AnsBool: proto.Bool(false),
	}
	pts, hasAuto = GetAutomaticPointsForAnswer(boolQn, ansFalse)
	if hasAuto || pts != 0 {
		t.Errorf("False answer want pts=0, hasAuto=false; got pts=%d, hasAuto=%v", pts, hasAuto)
	}

	// Text Question (not MCQ or BOOL) -> 0 pts, hasAuto = false
	textQn := &model.Question{
		Type: model.AnswerType_TEXT_ANSWER.Enum(),
	}
	ansText := &model.Answer{
		Type:    model.AnswerType_TEXT_ANSWER.Enum(),
		AnsText: proto.String("hello"),
	}
	pts, hasAuto = GetAutomaticPointsForAnswer(textQn, ansText)
	if hasAuto || pts != 0 {
		t.Errorf("Text answer want pts=0, hasAuto=false; got pts=%d, hasAuto=%v", pts, hasAuto)
	}
}
