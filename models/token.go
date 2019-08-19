package models

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	mrand "math/rand"
	"time"
)

// Token authorization token model
type Token struct {
	Key     string       `gorm:"type:varchar(40);PRIMARY_KEY;not null"`
	User    *UserProfile `gorm:"ForeignKey:UserID;SAVE_ASSOCIATIONS:false" json:"-"` //所属用户
	UserID  uint         `gorm:"column:user_id;index:idx_user_id;" json:"user_id"`   //所属用户id
	Created TypeJSONTime `gorm:"column:created;type:datetime;" json:"created_time"`
}

// NewToken return a token
func NewToken(user *UserProfile) *Token {

	t := &Token{
		User:    user,
		UserID:  user.ID,
		Created: JSONTimeNow(),
	}
	t.Key = t.generateKey()
	return t
}

// TableName Set Bucket's table name
func (t Token) TableName() string {
	return "authtoken_token"
}

func (t *Token) generateKey() string {

	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return getRandomString(40)
	}

	b2 := UintToBytes(t.UserID)
	b = append(b, b2...)
	return hex.EncodeToString(b)
}

// UintToBytes convert uint to bytes
func UintToBytes(value uint) []byte {

	var buf = make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(value))
	return buf
}

func getRandomString(length int) string {

	allowedChars := []byte("abcdefghijklmnopqrstuvwxyz0123456789")
	s := []byte{}
	r := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	for i := 0; i < length; i++ {
		s = append(s, allowedChars[r.Intn(len(allowedChars))])
	}
	return string(s)
}
