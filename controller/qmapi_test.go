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
