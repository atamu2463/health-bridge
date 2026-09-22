package model

const (
	ConditionCodeExcellent = "excellent"
	ConditionCodeGood      = "good"
	ConditionCodeNormal    = "normal"
	ConditionCodeCaution   = "caution"
	ConditionCodeBad       = "bad"
)

type Condition struct {
	ID           uint   `gorm:"primaryKey"`
	Code         string `gorm:"type:varchar;not null;uniqueIndex:ux_conditions_code"`
	Name         string `gorm:"type:varchar;not null;uniqueIndex:ux_conditions_name"`
	Score        int16  `gorm:"not null;uniqueIndex:ux_conditions_score"`
	DisplayOrder int16  `gorm:"not null;uniqueIndex:ux_conditions_display_order"`
}
