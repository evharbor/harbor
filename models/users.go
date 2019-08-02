package models

import (
	"harbor/utils/auth"
	"time"
)

const (
	// RoleNormal = 0; 普通用户
	RoleNormal TypeRole = 0
	// RoleSuperUser = 1;超级用户
	RoleSuperUser TypeRole = 1 << 0
	// RoleAppSuperUser = 2; 第三方APP超级用户,有权限获取普通用户安全凭证
	RoleAppSuperUser TypeRole = 1 << 1
	// RoleStaff = 4;职员，可登录后台管理
	RoleStaff TypeRole = 1 << 2
	// RoleStaffSuperUser RoleSuperUser and RoleStaff
	RoleStaffSuperUser TypeRole = RoleStaff | RoleSuperUser
)

// TypeRole role type
type TypeRole int16

// Value return int16
func (r TypeRole) Value() int16 {

	return int16(r)
}

// UserProfile model
type UserProfile struct {
	ID          uint         `gorm:"primary_key" json:"id"`
	Username    string       `gorm:"type:varchar(150);unique_index:uidx_name;not null" json:"username,omitempty"`
	Password    string       `gorm:"type:varchar(128)" json:"-"`
	IsSuperUser bool         `gorm:"column:is_superuser;default:false;not null"  json:"-"`
	IsStaff     bool         `gorm:"column:is_staff;default:false;not null"  json:"-"`
	IsActive    bool         `gorm:"column:is_active;default:false;not null"  json:"-"`
	FirstName   string       `gorm:"column:first_name;type:varchar(30)" json:"first_name,omitempty"`
	LastName    string       `gorm:"column:last_name;type:varchar(150)" json:"last_name,omitempty"`
	Email       string       `gorm:"type:varchar(254);not null"`
	DateJoined  TypeJSONTime `gorm:"column:date_joined;type:datetime;not null"  json:"-"`
	LastLogin   TypeJSONTime `gorm:"column:last_login;type:datetime"  json:"-"`
	Company     string       `gorm:"type:varchar(255)"`
	Telephone   string       `gorm:"type:varchar(11)"`
	ThirdApp    uint         `gorm:"not null;default:0" json:"-"`
	SecretKey   string       `gorm:"type:varchar(20)" json:"-"`
	LastActive  TypeJSONTime `gorm:"index;type:date"  json:"-"`
	Role        int16        `gorm:"type:smallint" json:"-"`
}

// TableName Set UserProfile's table name
func (UserProfile) TableName() string {
	return "users_userprofile"
}

// NewUserProfile create a user object by default
func NewUserProfile() *UserProfile {
	return &UserProfile{
		DateJoined: TypeJSONTime{Time: time.Now()},
	}
}

// IsActived return true if user is active,otherwise false
func (u UserProfile) IsActived() bool {

	return u.IsActive
}

// CheckPassword Check whether the user's password is the same as the given password
func (u UserProfile) CheckPassword(pw string) bool {

	return auth.CheckPassword(pw, u.Password)
}

// SetPassword set new password
func (u *UserProfile) SetPassword(pw string) {

	u.Password = auth.MakePassword(pw)
	return
}

// UsernameColumnName Return the table field name corresponding to the username field
func (u UserProfile) UsernameColumnName() string {

	return "username"
}

// IsSuper return true if user is super user
func (u UserProfile) IsSuper() bool {

	return u.IsSuperUser
}

// IsStaffUser return true if user is staff user
func (u UserProfile) IsStaffUser() bool {

	return u.IsStaff
}

// IsAppSuperUser return true if user is third app super user
func (u UserProfile) IsAppSuperUser() bool {

	return u.IsRole(RoleAppSuperUser)
}

// IsNormalUser return true if user is normal user
func (u UserProfile) IsNormalUser() bool {

	if u.IsSuper() {
		return false
	}
	if u.IsStaffUser() {
		return false
	}
	if u.IsAppSuperUser() {
		return false
	}

	return true
}

// IsRole check whether user's role is input role
func (u UserProfile) IsRole(role TypeRole) bool {

	if role == RoleNormal {
		return u.Role == role.Value()
	}

	if (u.Role & int16(role)) != 0 {
		return true
	}

	return false
}

// SetRole set user's role is input role
func (u *UserProfile) SetRole(role TypeRole) {

	if (role & RoleSuperUser) != 0 {
		u.IsSuperUser = true
	} else {
		u.IsSuperUser = false
	}

	if (role & RoleStaff) != 0 {
		u.IsStaff = true
	} else {
		u.IsStaff = false
	}

	u.Role = role.Value()
	return
}

// AddRole add input role for user
func (u *UserProfile) AddRole(role TypeRole) {

	// 普通用户角色不能和其他角色同时存在
	if role == RoleNormal {
		u.SetRole(RoleNormal)
		return
	}

	if (role & RoleSuperUser) != 0 {
		u.IsSuperUser = true
	}

	if (role & RoleStaff) != 0 {
		u.IsStaff = true
	}

	u.Role = u.Role | role.Value()
	return
}
