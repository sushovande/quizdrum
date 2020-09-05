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

// GormQuiz is the persisted version of the Quiz proto
type GormQuiz struct {
	gorm.Model
	// ProtoData contains the serialized Quiz proto
	ProtoData []byte
	// GormQuestions are the persisted questions in this quiz
	GormQuestions []GormQuestion
}

// GetQuiz returns a Quiz with the given ID, with questions pre-populated
func (p *Persistence) GetQuiz(id int64) (*Quiz, error) {
	var gq GormQuiz
	if err := p.db.Preload("GormQuestions").First(&gq, id).Error; err != nil {
		return nil, err
	}
	return getQuizFromGormQuiz(&gq)
}

// GetQuizWithoutQuestions returns a Quiz with the given ID, without questions pre-populated
func (p *Persistence) GetQuizWithoutQuestions(id int64) (*Quiz, error) {
	var gq GormQuiz
	if err := p.db.First(&gq, id).Error; err != nil {
		return nil, err
	}
	return getQuizFromGormQuiz(&gq)
}

// GetQuizFromQuestionID returns a Quiz that contains this question ID.
// Other questions are not pre-populated in the Quiz.
func (p *Persistence) GetQuizFromQuestionID(id uint) (*Quiz, error) {
	var gq GormQuiz
	var qn GormQuestion
	if err := p.db.First(&qn, id).Error; err != nil {
		return nil, err
	}
	if err := p.db.First(&gq, qn.GormQuizID).Error; err != nil {
		return nil, err
	}
	q, err := getQuizFromGormQuiz(&gq)
	if err != nil {
		return nil, err
	}
	return q, nil
}

// CreateQuiz stores a quiz object to disk and returns the ID.
// The questions belonging to this quiz must be persisted separately.
// An ACL entry is also created allowing the creator write privileges.
func (p *Persistence) CreateQuiz(q *Quiz) (uint, error) {
	var resultingQuizID uint
	err := p.db.Transaction(func(tx *gorm.DB) error {
		if len(q.Quizmasters) == 0 || q.Quizmasters[0].GetUserId() == 0 {
			return fmt.Errorf("a quiz can only be created by a logged in user")
		}
		q1 := proto.Clone(q).(*Quiz)
		q1.Questions = nil
		b, err := proto.Marshal(q1)
		if err != nil {
			return err
		}
		gq := GormQuiz{ProtoData: b}
		err = tx.Create(&gq).Error
		if err != nil {
			return err
		}
		resultingQuizID = gq.ID
		initACL := AccessType{ReadAllowed: proto.Bool(true), WriteAllowed: proto.Bool(true)}
		bacl, err := proto.Marshal(&initACL)
		if err != nil {
			return err
		}
		gacl := GormAccessControl{
			QuizID:    resultingQuizID,
			UserID:    uint(q.Quizmasters[0].GetUserId()),
			ProtoData: bacl,
		}
		if err = tx.Create(&gacl).Error; err != nil {
			return err
		}
		return nil
	})
	return resultingQuizID, err
}

// SaveQuiz stores a quiz object to disk.
func (p *Persistence) SaveQuiz(q *Quiz) error {
	q1 := proto.Clone(q).(*Quiz)
	q1.Questions = nil
	gq, err := getGormQuizFromQuiz(q1)
	if err != nil {
		return err
	}
	return p.db.Save(&gq).Error
}

// DeleteQuiz soft-deletes a quiz
func (p *Persistence) DeleteQuiz(qzid int64) error {
	var gq GormQuiz
	gq.ID = uint(qzid)
	return p.db.Delete(&gq).Error
}

// ReinstateQuiz un-deletes a quiz
func (p *Persistence) ReinstateQuiz(qzid int64) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		var gq GormQuiz
		if err := tx.Unscoped().First(&gq, uint(qzid)).Error; err != nil {
			return err
		}
		gq.DeletedAt.Valid = false
		return tx.Unscoped().Save(&gq).Error
	})
}

// SaveQuizMetadata stores the metadata of the quiz without affecting other fields.
func (p *Persistence) SaveQuizMetadata(qz *Quiz) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		var gq GormQuiz
		if err := tx.First(&gq, uint(qz.GetId())).Error; err != nil {
			return err
		}
		qzo, err := getQuizFromGormQuiz(&gq)
		if err != nil {
			return err
		}
		qzo.Title = proto.String(qz.GetTitle())
		qzo.HtmlDescription = proto.String(qz.GetHtmlDescription())
		gq2, err := getGormQuizFromQuiz(qzo)
		if err != nil {
			return nil
		}
		return tx.Save(gq2).Error
	})
}

// GetAllQuizzes returns all the quizzes in the database (without questions populated)
func (p *Persistence) GetAllQuizzes() ([]*Quiz, error) {
	gqs := make([]GormQuiz, 0)
	if err := p.db.Find(&gqs).Error; err != nil {
		return nil, err
	}
	qs := make([]*Quiz, 0)
	for _, gq := range gqs {
		q, err := getQuizFromGormQuiz(&gq)
		if err != nil {
			return nil, err
		}
		qs = append(qs, q)
	}
	return qs, nil
}

// RegisterParticipant adds a user to the quiz as a participant if needed
// This is an atomic read-modify-write of the quiz proto
func (p *Persistence) RegisterParticipant(qid int64, userID int64, profileName string) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		var gq GormQuiz
		if err := tx.First(&gq, qid).Error; err != nil {
			return err
		}
		qp, err := getQuizFromGormQuiz(&gq)
		if err != nil {
			return err
		}
		var pp *ParticipantProfile
		for _, v := range qp.GetParticipants() {
			if v.GetUserId() == userID {
				pp = v
				break
			}
		}

		if pp == nil {
			pp = &ParticipantProfile{}
			pp.UserId = proto.Int64(int64(userID))
			qp.Participants = append(qp.Participants, pp)
		}

		pp.ProfileName = proto.String(profileName)
		pp.CompletedRegistration = proto.Bool(true)
		b, err := proto.Marshal(qp)
		if err != nil {
			return err
		}
		gq.ProtoData = b
		err = tx.Save(&gq).Error
		if err != nil {
			return err
		}
		return nil
	})
}

func getQuizFromGormQuiz(gq *GormQuiz) (*Quiz, error) {
	var q Quiz
	if err := proto.Unmarshal(gq.ProtoData, &q); err != nil {
		return nil, err
	}
	// Load the values from the other columns that might not
	// have been persisted in the proto.
	q.Id = proto.Int64(int64(gq.ID))
	for _, qn := range gq.GormQuestions {
		var qp Question
		if err := proto.Unmarshal(qn.ProtoData, &qp); err != nil {
			return nil, err
		}
		qp.Id = proto.Int64(int64(qn.ID))
		qp.QuizId = proto.Int64(int64(qn.GormQuizID))
		q.Questions = append(q.Questions, &qp)
	}
	return &q, nil
}

func getGormQuizFromQuiz(qz *Quiz) (*GormQuiz, error) {
	b, err := proto.Marshal(qz)
	if err != nil {
		return nil, err
	}
	gq := GormQuiz{ProtoData: b}
	gq.ID = uint(qz.GetId())
	return &gq, nil
}
