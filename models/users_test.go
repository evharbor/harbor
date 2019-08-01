package models_test

import (
	"harbor/models"
	"testing"
)

func TestRole(t *testing.T) {

	user := models.UserProfile{}
	if user.IsRole(models.RoleNormal) == false {
		t.Errorf("IsRole should be RoleNormal")
	}
	if user.IsRole(models.RoleSuperUser) == true {
		t.Errorf("IsRole should not  be RoleSuperUser")
	}

	user.AddRole(models.RoleSuperUser)
	user.AddRole(models.RoleStaff)
	if user.IsRole(models.RoleNormal) {
		t.Errorf("IsRole should not be RoleNormal")
	}
	if user.IsRole(models.RoleSuperUser) != true {
		t.Errorf("IsRole should be RoleSuperUser")
	}
	if user.IsRole(models.RoleStaff) != true {
		t.Errorf("IsRole should be RoleStaff")
	}

	user.SetRole(models.RoleAppSuperUser)
	if user.IsRole(models.RoleNormal) {
		t.Errorf("IsRole should not be RoleNormal")
	}
	if user.IsRole(models.RoleSuperUser) == true {
		t.Errorf("IsRole should not be RoleSuperUser")
	}
	if user.IsRole(models.RoleStaff) == true {
		t.Errorf("IsRole should not be RoleStaff")
	}

	user.SetRole(models.RoleNormal)
	if user.IsRole(models.RoleNormal) == false {
		t.Errorf("IsRole should be RoleNormal")
	}
	if user.IsRole(models.RoleAppSuperUser) == true {
		t.Errorf("IsRole should not be RoleAppSuperUser")
	}
	t.Log("IsRole test ok")
}
