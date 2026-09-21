package model

const (
	RoleNameManager  = "manager"
	RoleNameEmployee = "employee"
)

type Role struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"type:varchar;not null;uniqueIndex:ux_roles_name"`
}
