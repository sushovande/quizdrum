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
	"net/http"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// GormUser represents a user of the site who has either
// logged in via Google or logged in as Guest.
type GormUser struct {
	gorm.Model
	// GoogleID is the `sub` field from the Google response
	// If empty, it indicates a guest login.
	GoogleID string `sql:"index"`
	// Cookies are the cookies associated with this user.
	Cookies []GormCookie
	// ProtoData contains a serialized User proto.
	ProtoData []byte
}

// GormCookie represents a mapping from a cookie to a user
type GormCookie struct {
	// ID is the content of the cookie. It is a primary key.
	ID string
	// The User ID to which this cookie belongs. It is a foreign key to a `GormUser`.
	GormUserID uint
	// Expiry is the unix timestamp in seconds at which this cookie expires
	Expiry int64
}

// GormAccessControl represents a mapping from a (user, quiz) pair to an access type
type GormAccessControl struct {
	// UserID is the ID of the user who has access.
	UserID uint
	// QuizID is the entity to which the user has access.
	QuizID uint
	// ProtoData contains the serialized AccessType proto.
	ProtoData []byte
}

// NewGuestLogin creates a new user and associates it with the given cookie.
// Returns the ID of the user that was just created.
func (p *Persistence) NewGuestLogin(ck string, exp int64) (uint, error) {
	var gu GormUser
	err := p.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&gu).Error; err != nil {
			return err
		}
		gc := &GormCookie{
			ID:         ck,
			GormUserID: gu.ID,
			Expiry:     exp,
		}
		if err := tx.Create(&gc).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return gu.ID, nil
}

// guserProtoContainer is a wrapper on top of the GUser proto
// that satisfies encoding/json.Unmarshaler, which the jws.Claims
// call depends upon.
type guserProtoContainer struct {
	g *GUser
}

func (f *guserProtoContainer) UnmarshalJSON(b []byte) error {
	f.g = &GUser{}
	return protojson.Unmarshal(b, f.g)
}

// OAuthLoginFromToken parses the token and returns existing OAuth user if found,
// or creates a new user if not found.
func (p *Persistence) OAuthLoginFromToken(idToken string) (*User, error) {
	if len(idToken) == 0 {
		return nil, fmt.Errorf("empty idToken")
	}
	jws, err := jwt.ParseSigned(idToken, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil {
		return nil, err
	}

	kid := jws.Headers[0].KeyID
	gc, err := p.GetCertWithID(kid)
	if err != nil {
		return nil, err
	}

	var fp guserProtoContainer
	if err = jws.Claims(gc, &fp); err != nil {
		return nil, err
	}

	g := fp.g
	if g.GetAud() != p.OAuthClientID {
		return nil, fmt.Errorf("wrong client ID")
	}

	// At this point we are confident that `g` is a valid GUser.
	// Now, we upsert this.
	u, err := p.GetUserFromGoogleID(g.GetSub())
	if err != nil {
		return nil, err
	}

	if u == nil {
		var pu User
		pu.GoogleUser = proto.Clone(g).(*GUser)
		gu, err := getGormUserFromUser(&pu)
		if err != nil {
			return nil, err
		}
		if err = p.db.Create(&gu).Error; err != nil {
			return nil, err
		}
		pu.Id = proto.Int64(int64(gu.ID))
		return &pu, nil
	}

	// This Google user already exists.
	// Update the google proto and return
	u.GoogleUser = proto.Clone(g).(*GUser)
	gu, err := getGormUserFromUser(u)
	if err != nil {
		return nil, err
	}
	if err = p.db.Save(&gu).Error; err != nil {
		return nil, err
	}
	return u, nil
}

// NewCookieForUser adds a cookie for a user. This will typically be used
// for Google users only.
func (p *Persistence) NewCookieForUser(ck string, id uint, exp int64) error {
	gc := &GormCookie{
		ID:         ck,
		GormUserID: id,
		Expiry:     exp,
	}
	return p.db.Create(&gc).Error
}

// GetUserFromGoogleID gets a User from the Google ID that was set.
// Returns nil, nil if record is not found.
func (p *Persistence) GetUserFromGoogleID(sub string) (*User, error) {
	var u GormUser
	u.GoogleID = sub
	if err := p.db.Where(&u).First(&u).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return getUserFromGormUser(&u)
}

// GetUserFromCookie gets a User from the cookie that was set
func (p *Persistence) GetUserFromCookie(ck string) (*User, error) {
	var cgk GormCookie
	cgk.ID = ck
	if err := p.db.Where(&cgk).First(&cgk).Error; err != nil {
		return nil, err
	}
	if time.Now().Unix() > cgk.Expiry {
		return nil, fmt.Errorf("cookie expired")
	}
	var gu GormUser
	if err := p.db.First(&gu, cgk.GormUserID).Error; err != nil {
		return nil, err
	}
	return getUserFromGormUser(&gu)
}

// GetUserFromCookieAndError gets a User from the cookie, but checks the error first
// This is a convenience to chain the call from r.Cookie("sid")
func (p *Persistence) GetUserFromCookieAndError(ck *http.Cookie, err error) (*User, error) {
	if err != nil {
		return nil, err
	}
	return p.GetUserFromCookie(ck.Value)
}

// DeleteCookie deletes the cookie from the database
func (p *Persistence) DeleteCookie(ck string) error {
	var cgk GormCookie
	cgk.ID = ck
	if err := p.db.Delete(&cgk).Error; err != nil {
		return err
	}
	return nil
}

// ValidateWritePrivileges returns an error if the user does not have write access to the given quiz ID
func (p *Persistence) ValidateWritePrivileges(qzid int64, u *User) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		// first try the direct ACL
		gacl := GormAccessControl{
			QuizID: uint(qzid),
			UserID: uint(u.GetId()),
		}
		aclerr := tx.Where(&gacl).Take(&gacl).Error
		if aclerr == nil {
			var atype AccessType
			if err := proto.Unmarshal(gacl.ProtoData, &atype); err != nil {
				return err
			}
			if atype.GetWriteAllowed() {
				return nil
			}
			return fmt.Errorf("User is not allowed to write to this quiz")
		} else if aclerr == gorm.ErrRecordNotFound {
			// This path is only viable if the user is a Google-logged in user
			if u.GoogleUser.GetSub() == "" {
				return fmt.Errorf("non-creator guest users are not allowed to write to quizzes")
			}
			// fall back to reading the quiz
			var qz GormQuiz
			if err := tx.First(&qz, uint(qzid)).Error; err != nil {
				return err
			}
			q, err := getQuizFromGormQuiz(&qz)
			if err != nil {
				return nil
			}
			for _, qm := range q.GetQuizmasters() {
				if UserMatches(qm, u) {
					return nil
				}
			}
			return fmt.Errorf("User is not allowed to write to this quiz")
		}
		return aclerr
	})
}

// UserMatches tells us if the user matches some field of the quizmaster profile.
func UserMatches(qm *QuizmasterProfile, u *User) bool {
	if qm.GetGoogleSub() != "" {
		return qm.GetGoogleSub() == u.GoogleUser.GetSub()
	}
	if qm.GetUserId() != 0 {
		return qm.GetUserId() == u.GetId()
	}
	if qm.GetGoogleEmail() != "" {
		return qm.GetGoogleEmail() == u.GoogleUser.GetEmail()
	}
	// This should never happen
	return false
}

func getUserFromGormUser(gu *GormUser) (*User, error) {
	var u User
	if err := proto.Unmarshal(gu.ProtoData, &u); err != nil {
		return nil, err
	}
	u.Id = proto.Int64(int64(gu.ID))
	return &u, nil
}

func getGormUserFromUser(u *User) (*GormUser, error) {
	var gu GormUser
	if u.GetId() != 0 {
		gu.ID = uint(u.GetId())
	}
	if u.GoogleUser != nil && u.GoogleUser.GetSub() != "" {
		gu.GoogleID = u.GoogleUser.GetSub()
		b, err := proto.Marshal(u)
		if err != nil {
			return nil, err
		}
		gu.ProtoData = b
	}
	return &gu, nil
}
