package middlewares

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/jinzhu/gorm"
)

// CopyEmptyStruct Return a new type with the same structure as the input struct object
func CopyEmptyStruct(obj interface{}) (newObj interface{}) {

	var isPtr bool
	var utype reflect.Type
	var userv reflect.Value

	utype = reflect.TypeOf(obj)
	userv = reflect.ValueOf(obj)
	if userv.Kind() == reflect.Ptr {
		isPtr = true
		userv = userv.Elem()
		utype = utype.Elem()
	}

	uNew := reflect.New(utype)
	if !isPtr {
		uNew = uNew.Elem()
	}
	newObj = uNew.Interface()

	return
}

// Authenticate auth
func Authenticate(db *gorm.DB, username, password string, user interface{}) error {

	iu, ok := user.(IBasicAuth)
	if !ok {
		return errors.New("error to authenticate")
	}
	where := fmt.Sprintf("%s = ?", iu.UsernameColumnName())
	if r := db.Where(where, username).First(user); r.Error != nil {
		if r.RecordNotFound() {
			return errors.New("invalid username,user is not found")
		}
		s := fmt.Sprintf("select user error,%s", r.Error.Error())
		return errors.New(s)
	}

	// check actived user
	if !iu.IsActived() {
		return errors.New("user is not actived")
	}

	// check password
	if !iu.CheckPassword(password) {
		return errors.New("invalid password")
	}
	return nil
}
