package migration_test

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"backend/config"
	"backend/migration"
	"backend/model"

	"gorm.io/gorm"
)

var errRollbackTest = errors.New("テスト用トランザクションをロールバックします")

func TestMigrationOnPostgreSQL(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URLが未設定のためPostgreSQL統合テストをスキップします")
	}

	db, err := config.ConnectDB(databaseURL)
	if err != nil {
		t.Fatalf("テストDBへの接続に失敗しました: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("テストDBインスタンスの取得に失敗しました: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("テストDB接続の終了に失敗しました: %v", err)
		}
	})

	t.Run("migrationとseedを複数回実行できる", func(t *testing.T) {
		withMigratedDatabase(t, db, func(tx *gorm.DB) error {
			if err := migration.Run(tx); err != nil {
				return fmt.Errorf("2回目のmigration.Run()に失敗しました: %w", err)
			}
			if !tx.Migrator().HasConstraint(
				&model.HealthRecord{},
				"chk_health_records_comment_not_blank",
			) {
				return errors.New("health_records.commentのCHECK制約が作成されていません")
			}

			for _, table := range []any{
				&model.Role{},
				&model.User{},
				&model.Condition{},
				&model.HealthRecord{},
			} {
				if !tx.Migrator().HasTable(table) {
					return fmt.Errorf("テーブル %T が作成されていません", table)
				}
			}

			var roleCount int64
			if err := tx.Model(&model.Role{}).Count(&roleCount).Error; err != nil {
				return err
			}
			if roleCount != 2 {
				return fmt.Errorf("roles count = %d, want 2", roleCount)
			}

			var conditions []model.Condition
			if err := tx.Order("display_order").Find(&conditions).Error; err != nil {
				return err
			}
			if len(conditions) != 5 {
				return fmt.Errorf("conditions count = %d, want 5", len(conditions))
			}
			if conditions[0].Code != model.ConditionCodeExcellent ||
				conditions[4].Code != model.ConditionCodeBad {
				return fmt.Errorf("conditionsの表示順が正しくありません: %#v", conditions)
			}

			return nil
		})
	})

	t.Run("masterの一意制約", func(t *testing.T) {
		withMigratedDatabase(t, db, func(tx *gorm.DB) error {
			err := tx.Create(&model.Role{Name: model.RoleNameManager}).Error
			if err == nil {
				return errors.New("roles.nameの重複が拒否されませんでした")
			}
			return nil
		})
	})

	t.Run("userの外部キー制約", func(t *testing.T) {
		withMigratedDatabase(t, db, func(tx *gorm.DB) error {
			err := tx.Create(&model.User{
				Name:         "テスト利用者",
				Email:        "invalid-role@example.com",
				PasswordHash: "test-password-hash",
				RoleID:       999999,
			}).Error
			if err == nil {
				return errors.New("users.role_idの不正な参照が拒否されませんでした")
			}
			return nil
		})
	})

	t.Run("managerの自己参照外部キー制約", func(t *testing.T) {
		withMigratedDatabase(t, db, func(tx *gorm.DB) error {
			var employeeRole model.Role
			if err := tx.Where("name = ?", model.RoleNameEmployee).First(&employeeRole).Error; err != nil {
				return err
			}
			invalidManagerID := uint(999999)
			err := tx.Create(&model.User{
				Name:         "テスト従業員",
				Email:        "invalid-manager@example.com",
				PasswordHash: "test-password-hash",
				RoleID:       employeeRole.ID,
				ManagerID:    &invalidManagerID,
			}).Error
			if err == nil {
				return errors.New("users.manager_idの不正な参照が拒否されませんでした")
			}
			return nil
		})
	})

	t.Run("health recordの外部キー制約", func(t *testing.T) {
		withMigratedDatabase(t, db, func(tx *gorm.DB) error {
			var condition model.Condition
			if err := tx.Where("code = ?", model.ConditionCodeNormal).First(&condition).Error; err != nil {
				return err
			}
			err := tx.Create(&model.HealthRecord{
				EmployeeID:  999999,
				RecordDate:  time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC),
				Timing:      model.HealthRecordTimingClockIn,
				ConditionID: condition.ID,
				Comment:     "外部キー制約確認",
			}).Error
			if err == nil {
				return errors.New("health_records.employee_idの不正な参照が拒否されませんでした")
			}
			return nil
		})
	})

	t.Run("health recordのcondition外部キー制約", func(t *testing.T) {
		withMigratedDatabase(t, db, func(tx *gorm.DB) error {
			employee, _, err := createHealthRecordReferences(tx, "invalid-condition")
			if err != nil {
				return err
			}

			err = tx.Create(&model.HealthRecord{
				EmployeeID:  employee.ID,
				RecordDate:  time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC),
				Timing:      model.HealthRecordTimingClockIn,
				ConditionID: 999999,
				Comment:     "condition外部キー制約確認",
			}).Error
			if err == nil {
				return errors.New("health_records.condition_idの不正な参照が拒否されませんでした")
			}
			return nil
		})
	})

	t.Run("必須カラムのNOT NULL制約", func(t *testing.T) {
		withMigratedDatabase(t, db, func(tx *gorm.DB) error {
			requiredColumns := map[string][]string{
				"roles":          {"name"},
				"users":          {"name", "email", "password_hash", "role_id", "is_active", "created_at", "updated_at"},
				"conditions":     {"code", "name", "score", "display_order"},
				"health_records": {"employee_id", "record_date", "timing", "condition_id", "comment", "created_at"},
			}

			for table, columns := range requiredColumns {
				for _, column := range columns {
					var isNullable string
					if err := tx.Raw(`
						SELECT is_nullable
						FROM information_schema.columns
						WHERE table_schema = current_schema()
						  AND table_name = ?
						  AND column_name = ?
					`, table, column).Scan(&isNullable).Error; err != nil {
						return err
					}
					if isNullable != "NO" {
						return fmt.Errorf("%s.%s is_nullable = %q, want NO", table, column, isNullable)
					}
				}
			}

			return nil
		})
	})

	t.Run("health recordのtiming CHECK制約", func(t *testing.T) {
		withMigratedDatabase(t, db, func(tx *gorm.DB) error {
			employee, condition, err := createHealthRecordReferences(tx, "timing")
			if err != nil {
				return err
			}

			err = tx.Create(&model.HealthRecord{
				EmployeeID:  employee.ID,
				RecordDate:  time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC),
				Timing:      "invalid",
				ConditionID: condition.ID,
				Comment:     "CHECK制約確認",
			}).Error
			if err == nil {
				return errors.New("health_records.timingの不正値が拒否されませんでした")
			}
			return nil
		})
	})

	t.Run("health recordの複合一意制約", func(t *testing.T) {
		withMigratedDatabase(t, db, func(tx *gorm.DB) error {
			employee, condition, err := createHealthRecordReferences(tx, "unique")
			if err != nil {
				return err
			}

			record := model.HealthRecord{
				EmployeeID:  employee.ID,
				RecordDate:  time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC),
				Timing:      model.HealthRecordTimingClockIn,
				ConditionID: condition.ID,
				Comment:     "複合一意制約確認",
			}
			if err := tx.Create(&record).Error; err != nil {
				return err
			}

			record.ID = 0
			if err := tx.Create(&record).Error; err == nil {
				return errors.New("employee_id、record_date、timingの重複が拒否されませんでした")
			}
			return nil
		})
	})

	commentTests := []struct {
		name      string
		comment   string
		wantError bool
	}{
		{name: "通常コメント", comment: "体調は良好です"},
		{name: "文字vだけ", comment: "v"},
		{name: "空文字", comment: "", wantError: true},
		{name: "空白文字だけ", comment: " \t\n", wantError: true},
		{name: "垂直タブだけ", comment: "\v", wantError: true},
		{name: "500文字", comment: strings.Repeat("a", 500)},
		{name: "501文字", comment: strings.Repeat("a", 501), wantError: true},
	}

	for index, tt := range commentTests {
		t.Run("health recordのcomment制約/"+tt.name, func(t *testing.T) {
			withMigratedDatabase(t, db, func(tx *gorm.DB) error {
				employee, condition, err := createHealthRecordReferences(
					tx,
					fmt.Sprintf("comment-%d", index),
				)
				if err != nil {
					return err
				}

				err = tx.Create(&model.HealthRecord{
					EmployeeID:  employee.ID,
					RecordDate:  time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC),
					Timing:      model.HealthRecordTimingClockOut,
					ConditionID: condition.ID,
					Comment:     tt.comment,
				}).Error
				if tt.wantError && err == nil {
					return fmt.Errorf("health_records.commentの%sが拒否されませんでした", tt.name)
				}
				if !tt.wantError && err != nil {
					return fmt.Errorf("health_records.commentの%sが保存できませんでした: %w", tt.name, err)
				}
				return nil
			})
		})
	}
}

func withMigratedDatabase(t *testing.T, db *gorm.DB, verify func(*gorm.DB) error) {
	t.Helper()

	schemaName := fmt.Sprintf("issue87_test_%d", time.Now().UnixNano())
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`CREATE SCHEMA "` + schemaName + `"`).Error; err != nil {
			return fmt.Errorf("テスト用schemaの作成に失敗しました: %w", err)
		}
		if err := tx.Exec(`SET LOCAL search_path TO "` + schemaName + `"`).Error; err != nil {
			return fmt.Errorf("search_pathの設定に失敗しました: %w", err)
		}
		if err := migration.Run(tx); err != nil {
			return err
		}
		if err := verify(tx); err != nil {
			return err
		}
		return errRollbackTest
	})

	if !errors.Is(err, errRollbackTest) {
		t.Fatalf("PostgreSQL統合テストに失敗しました: %v", err)
	}
}

func createHealthRecordReferences(tx *gorm.DB, suffix string) (model.User, model.Condition, error) {
	var managerRole model.Role
	if err := tx.Where("name = ?", model.RoleNameManager).First(&managerRole).Error; err != nil {
		return model.User{}, model.Condition{}, err
	}
	var employeeRole model.Role
	if err := tx.Where("name = ?", model.RoleNameEmployee).First(&employeeRole).Error; err != nil {
		return model.User{}, model.Condition{}, err
	}

	manager := model.User{
		Name:         "テスト管理者",
		Email:        "manager-" + suffix + "@example.com",
		PasswordHash: "test-password-hash",
		RoleID:       managerRole.ID,
	}
	if err := tx.Create(&manager).Error; err != nil {
		return model.User{}, model.Condition{}, err
	}

	employee := model.User{
		Name:         "テスト従業員",
		Email:        "employee-" + suffix + "@example.com",
		PasswordHash: "test-password-hash",
		RoleID:       employeeRole.ID,
		ManagerID:    &manager.ID,
	}
	if err := tx.Create(&employee).Error; err != nil {
		return model.User{}, model.Condition{}, err
	}

	var condition model.Condition
	if err := tx.Where("code = ?", model.ConditionCodeNormal).First(&condition).Error; err != nil {
		return model.User{}, model.Condition{}, err
	}

	return employee, condition, nil
}
