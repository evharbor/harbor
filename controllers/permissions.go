package controllers

import (
	"harbor/models"
)

// IsSuperUser check whether has super user permission
func IsSuperUser(user *models.UserProfile) bool {

	return user.IsSuper()
}

// IsStaffUser check whether has staff user permission
func IsStaffUser(user *models.UserProfile) bool {

	return user.IsStaffUser()
}

// IsAppSuperUser check whether has third app super user permission
func IsAppSuperUser(user *models.UserProfile) bool {

	return user.IsAppSuperUser()
}
