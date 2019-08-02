package controllers

import (
	"harbor/models"
)

// IsAuthenticatedUser check whether user is authenticated
func IsAuthenticatedUser(user *models.UserProfile) bool {

	return user != nil
}

// IsSuperUser check whether has super user permission
func IsSuperUser(user *models.UserProfile) bool {
	if !IsAuthenticatedUser(user) {
		return false
	}
	return user.IsSuper()
}

// IsStaffUser check whether has staff user permission
func IsStaffUser(user *models.UserProfile) bool {

	if !IsAuthenticatedUser(user) {
		return false
	}
	return user.IsStaffUser()
}

// IsAppSuperUser check whether has third app super user permission
func IsAppSuperUser(user *models.UserProfile) bool {

	if !IsAuthenticatedUser(user) {
		return false
	}
	return user.IsAppSuperUser()
}

// IsStaffSuperUser check whether has super and staff user permission
func IsStaffSuperUser(user *models.UserProfile) bool {
	if !IsAuthenticatedUser(user) {
		return false
	}
	return user.IsSuper() && user.IsStaffUser()
}
