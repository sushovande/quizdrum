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

package model

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
)

func TestQuizBasicOperations(t *testing.T) {
	var p Persistence
	if err := p.Initialize(":memory:", "oauth_client_fake_id"); err != nil {
		t.Fatal(err)
	}

	uid, err := p.NewGuestLogin("cookie-cookie-re", time.Now().Unix()+10000)
	if err != nil {
		t.Fatalf("failed to log in. %v", err)
	}

	var qz Quiz
	qz.Title = proto.String("qztitle")
	qz.HtmlDescription = proto.String("qzdescr")
	qz.Quizmasters = append(qz.Quizmasters, &QuizmasterProfile{UserId: proto.Int64(int64(uid))})

	qzid, err := p.CreateQuiz(&qz)
	if err != nil {
		t.Fatalf("could not create quiz. %v", err)
	}

	if err := p.DeleteQuiz(int64(qzid)); err != nil {
		t.Errorf("could not delete quiz. %v", err)
	}

	qzs, err := p.GetAllQuizzes()
	if err != nil {
		t.Fatalf("could not fetch all quizzes. %v", err)
	}
	if ln := len(qzs); ln != 0 {
		t.Errorf("Got some quiz even after delete. want: 0, got: %v", ln)
	}

	if err := p.ReinstateQuiz(int64(qzid)); err != nil {
		t.Errorf("could not reinstate quiz. %v", err)
	}

	qzs, err = p.GetAllQuizzes()
	if err != nil {
		t.Fatalf("could not fetch all quizzes. %v", err)
	}
	if ln := len(qzs); ln != 1 {
		t.Fatalf("Got no quiz even after reinstate. want: 1, got: %v", ln)
	}
	if ti := qzs[0].GetTitle(); ti != qz.GetTitle() {
		t.Errorf("wrong title. want %v, got %v.", qz.GetTitle(), ti)
	}
}
