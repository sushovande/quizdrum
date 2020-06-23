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
	"github.com/jinzhu/gorm"
	"google.golang.org/protobuf/proto"
)

// GormQuestion is the persisted version of the Question proto
type GormQuestion struct {
	gorm.Model
	// GormQuizID is the quiz id to which this question belongs.
	// Foreign key reference to the GormQuiz table.
	GormQuizID uint
	// Answers is the set of responses to this question.
	Answers []GormAnswer
	// ProtoData contains the serialized Question proto
	ProtoData []byte
}

// CreateQuestion appends a new question to the quiz. The QuizId field
// must be populated for this to succeed. The ID field is ignored.
func (p *Persistence) CreateQuestion(qp *Question) (uint, error) {
	var qn GormQuestion
	qn.GormQuizID = uint(qp.GetQuizId())
	b, err := proto.Marshal(qp)
	if err != nil {
		return 0, err
	}
	qn.ProtoData = b
	if err = p.db.Create(&qn).Error; err != nil {
		return 0, err
	}
	return qn.ID, nil
}

// SaveQuestion updates the question in the DB. The question ID must be set.
func (p *Persistence) SaveQuestion(qp *Question) error {
	var qn GormQuestion
	qn.ID = uint(qp.GetId())
	qn.GormQuizID = uint(qp.GetQuizId())
	b, err := proto.Marshal(qp)
	if err != nil {
		return err
	}
	qn.ProtoData = b
	return p.db.Save(&qn).Error
}

// GetQuestionByID returns the question proto given an ID.
func (p *Persistence) GetQuestionByID(qnid uint) (*Question, error) {
	var gqn GormQuestion
	if err := p.db.First(&gqn, qnid).Error; err != nil {
		return nil, err
	}
	return getQuestionFromGormQuestion(&gqn)
}

// DeleteQuestion performs a soft delete of the question
func (p *Persistence) DeleteQuestion(qid uint) error {
	var qn GormQuestion
	qn.ID = qid
	return p.db.Delete(&qn).Error
}

func getQuestionFromGormQuestion(gqn *GormQuestion) (*Question, error) {
	var qn Question
	if err := proto.Unmarshal(gqn.ProtoData, &qn); err != nil {
		return nil, err
	}
	qn.Id = proto.Int64(int64(gqn.ID))
	return &qn, nil
}
