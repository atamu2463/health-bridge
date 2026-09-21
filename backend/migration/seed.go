package migration

import (
	"fmt"

	"backend/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func seedMasterData(db *gorm.DB) error {
	roles := []model.Role{
		{Name: model.RoleNameManager},
		{Name: model.RoleNameEmployee},
	}
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoNothing: true,
	}).Create(&roles).Error; err != nil {
		return fmt.Errorf("rolesマスターデータの投入に失敗しました: %w", err)
	}

	conditions := []model.Condition{
		{Code: model.ConditionCodeExcellent, Name: "とても良好", Score: 2, DisplayOrder: 1},
		{Code: model.ConditionCodeGood, Name: "良好", Score: 1, DisplayOrder: 2},
		{Code: model.ConditionCodeNormal, Name: "普通", Score: 0, DisplayOrder: 3},
		{Code: model.ConditionCodeCaution, Name: "注意", Score: -1, DisplayOrder: 4},
		{Code: model.ConditionCodeBad, Name: "悪化", Score: -2, DisplayOrder: 5},
	}
	if err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name",
			"score",
			"display_order",
		}),
	}).Create(&conditions).Error; err != nil {
		return fmt.Errorf("conditionsマスターデータの投入に失敗しました: %w", err)
	}

	return nil
}
