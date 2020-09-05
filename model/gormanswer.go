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
	"fmt"

	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// GormAnswer is the persisted version of the Answer proto
type GormAnswer struct {
	gorm.Model
	// GormQuestionID is the question to which this is an answer.
	// It can be thought of as a foreign key to the Questions table
	GormQuestionID uint
	// GormUserID is the user who solved this question.
	// It can be thought of as a foreign key to the Users table.
	GormUserID uint
	// ProtoData contains the serialized Answer proto
	ProtoData []byte
}

// CreateAnswer stores a new answer to a question in the db
func (p *Persistence) CreateAnswer(ans *Answer) (uint, error) {
	var ansid uint
	txerr := p.db.Transaction(func(tx *gorm.DB) error {
		var ga GormAnswer
		ga.GormQuestionID = uint(ans.GetQuestionId())
		ga.GormUserID = uint(ans.GetSolverId())

		// Check if the answer by this user, qn pair already exists
		// If the query succeeds, the ga.ID field is set
		err := tx.Where(&ga).Take(&ga).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		if ga.ID != 0 {
			oldAns, err := getAnswerFromGormAnswer(ga)
			if err != nil {
				return err
			}
			ans.PointsAwarded = proto.Int64(oldAns.GetPointsAwarded())
		}

		b, err := proto.Marshal(ans)
		if err != nil {
			return err
		}
		ga.ProtoData = b

		if ga.ID != 0 {
			if err = tx.Save(&ga).Error; err != nil {
				return err
			}
		} else {
			if err = tx.Create(&ga).Error; err != nil {
				return err
			}
		}
		ansid = ga.ID
		return nil
	})
	return ansid, txerr
}

// UpdateAnswer stores an updated answer in the db (Note: scores will be preserved)
func (p *Persistence) UpdateAnswer(ans *Answer) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		var ga GormAnswer
		if err := tx.First(&ga, uint(ans.GetId())).Error; err != nil {
			return err
		}
		oldAns, err := getAnswerFromGormAnswer(ga)
		if err != nil {
			return err
		}
		if oldAns.GetSolverId() != ans.GetSolverId() {
			return fmt.Errorf(
				"unauthorized update qnid: %v belongs to %v and not current user %v",
				oldAns.GetId(), oldAns.GetSolverId(), ans.GetSolverId())
		}
		ans.PointsAwarded = proto.Int64(oldAns.GetPointsAwarded())
		nga, err := getGormAnswerFromAnswer(ans)
		if err != nil {
			return err
		}
		return tx.Save(nga).Error
	})
}

// GetAnswerByID fetches an answer by a given ID
func (p *Persistence) GetAnswerByID(id uint) (*Answer, error) {
	var ga GormAnswer
	err := p.db.First(&ga, id).Error
	if err != nil {
		return nil, err
	}
	return getAnswerFromGormAnswer(ga)
}

// GetAnswerByUserAndQuestion gets the answer provided by the user for the given question
func (p *Persistence) GetAnswerByUserAndQuestion(u *User, qn *Question) (*Answer, error) {
	var ga GormAnswer
	ga.GormQuestionID = uint(qn.GetId())
	ga.GormUserID = uint(u.GetId())
	if err := p.db.Where(&ga).Take(&ga).Error; err != nil {
		return nil, err
	}
	return getAnswerFromGormAnswer(ga)
}

// GetAllAnswersToQuestionID fetches all the answers to a given question ID
func (p *Persistence) GetAllAnswersToQuestionID(id uint) ([]*Answer, error) {
	gas := make([]GormAnswer, 0)
	err := p.db.Find(&gas, "gorm_question_id = ?", id).Error
	if err != nil {
		return nil, err
	}
	sansa := make([]*Answer, 0)
	for _, ga := range gas {
		s, err := getAnswerFromGormAnswer(ga)
		if err != nil {
			return nil, err
		}
		sansa = append(sansa, s)
	}
	return sansa, nil
}

// SaveMultipleAnswers saves multiple answers to the db. This is useful when
// storing scores, for example.
func (p *Persistence) SaveMultipleAnswers(sansa []*Answer) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		for _, ans := range sansa {
			ga, err := getGormAnswerFromAnswer(ans)
			if err != nil {
				return err
			}
			if err := tx.Save(ga).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetAllAnswersForSetOfQuestions gets all the answers for a set of questions.
// This can be used to get all the answers for a quiz, for example, to build a scoreboard.
func (p *Persistence) GetAllAnswersForSetOfQuestions(qns []*Question) (map[*Question][]*Answer, error) {
	board := make(map[*Question][]*Answer)
	for _, qn := range qns {
		var ga []GormAnswer
		if err := p.db.Find(&ga, "gorm_question_id = ?", uint(qn.GetId())).Error; err != nil {
			return nil, err
		}
		sansa := make([]*Answer, 0, len(ga))
		for _, g := range ga {
			a, err := getAnswerFromGormAnswer(g)
			if err != nil {
				return nil, err
			}
			sansa = append(sansa, a)
		}
		board[qn] = sansa
	}
	return board, nil
}

func getAnswerFromGormAnswer(ga GormAnswer) (*Answer, error) {
	var ans Answer
	if err := proto.Unmarshal(ga.ProtoData, &ans); err != nil {
		return nil, err
	}
	ans.Id = proto.Int64(int64(ga.ID))
	return &ans, nil
}

func getGormAnswerFromAnswer(ans *Answer) (*GormAnswer, error) {
	var ga GormAnswer
	ga.GormQuestionID = uint(ans.GetQuestionId())
	ga.GormUserID = uint(ans.GetSolverId())
	ga.ID = uint(ans.GetId())
	b, err := proto.Marshal(ans)
	if err != nil {
		return nil, err
	}
	ga.ProtoData = b
	return &ga, nil
}
