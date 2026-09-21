package model

import "time"

type HealthRecordTiming string

const (
	HealthRecordTimingClockIn  HealthRecordTiming = "clockIn"
	HealthRecordTimingClockOut HealthRecordTiming = "clockOut"
)

type HealthRecord struct {
	ID          uint               `gorm:"primaryKey"`
	EmployeeID  uint               `gorm:"not null;uniqueIndex:ux_health_records_employee_date_timing,priority:1"`
	Employee    User               `gorm:"foreignKey:EmployeeID"`
	RecordDate  time.Time          `gorm:"type:date;not null;uniqueIndex:ux_health_records_employee_date_timing,priority:2"`
	Timing      HealthRecordTiming `gorm:"type:varchar;not null;check:chk_health_records_timing,timing IN ('clockIn','clockOut');uniqueIndex:ux_health_records_employee_date_timing,priority:3"`
	ConditionID uint               `gorm:"not null;index:idx_health_records_condition_id"`
	Condition   Condition          `gorm:"foreignKey:ConditionID"`
	Comment     string             `gorm:"type:varchar(500);not null"`
	CreatedAt   time.Time          `gorm:"not null;autoCreateTime"`
}
