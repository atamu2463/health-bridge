package model

import "time"

type User struct {
	ID            uint   `gorm:"primaryKey"`
	Name          string `gorm:"type:varchar;not null"`
	Email         string `gorm:"type:varchar;not null;uniqueIndex:ux_users_email"`
	PasswordHash  string `gorm:"type:varchar;not null"`
	RoleID        uint   `gorm:"not null;index:idx_users_role_id"`
	Role          Role   `gorm:"foreignKey:RoleID"`
	ManagerID     *uint  `gorm:"index:idx_users_manager_id"`
	Manager       *User  `gorm:"foreignKey:ManagerID"`
	IsActive      bool   `gorm:"not null;default:true"`
	DeactivatedAt *time.Time
	CreatedAt     time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt     time.Time `gorm:"not null;autoUpdateTime"`
}
