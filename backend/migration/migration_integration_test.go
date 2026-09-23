package migration_test

import (
	"crypto/sha256"
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
				&model.Session{},
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
				"sessions":       {"user_id", "token_digest", "expires_at", "created_at"},
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

	t.Run("sessionのuser外部キー制約", func(t *testing.T) {
		withMigratedDatabase(t, db, func(tx *gorm.DB) error {
			digest := sha256.Sum256([]byte(fmt.Sprintf("invalid-user-%d", time.Now().UnixNano())))
			err := tx.Create(&model.Session{
				UserID:      999999,
				TokenDigest: digest[:],
				ExpiresAt:   time.Now().Add(time.Hour),
			}).Error
			if err == nil {
				return errors.New("sessions.user_idの不正な参照が拒否されませんでした")
			}
			return nil
		})
	})

	t.Run("sessionのトークンダイジェスト制約", func(t *testing.T) {
		withMigratedDatabase(t, db, func(tx *gorm.DB) error {
			user, err := createSessionUser(tx, "digest")
			if err != nil {
				return err
			}
			digest := sha256.Sum256([]byte(fmt.Sprintf("duplicate-%d", time.Now().UnixNano())))
			session := model.Session{
				UserID:      user.ID,
				TokenDigest: digest[:],
				ExpiresAt:   time.Now().Add(time.Hour),
			}
			if err := tx.Create(&session).Error; err != nil {
				return err
			}

			session.ID = 0
			if err := tx.Create(&session).Error; err == nil {
				return errors.New("sessions.token_digestの重複が拒否されませんでした")
			}
			return nil
		})
	})

	t.Run("sessionのダイジェスト長CHECK制約", func(t *testing.T) {
		withMigratedDatabase(t, db, func(tx *gorm.DB) error {
			user, err := createSessionUser(tx, "digest-length")
			if err != nil {
				return err
			}
			err = tx.Create(&model.Session{
				UserID:      user.ID,
				TokenDigest: make([]byte, model.SessionTokenDigestSize-1),
				ExpiresAt:   time.Now().Add(time.Hour),
			}).Error
			if err == nil {
				return errors.New("32バイトでないsessions.token_digestが拒否されませんでした")
			}
			return nil
		})
	})

	t.Run("sessionのindexとカラム", func(t *testing.T) {
		withMigratedDatabase(t, db, func(tx *gorm.DB) error {
			for _, indexName := range []string{
				"ux_sessions_token_digest",
				"idx_sessions_user_id",
				"idx_sessions_expires_at",
			} {
				if !tx.Migrator().HasIndex(&model.Session{}, indexName) {
					return fmt.Errorf("sessionsのindex %s が作成されていません", indexName)
				}
			}
			if !tx.Migrator().HasConstraint(
				&model.Session{},
				"chk_sessions_token_digest_length",
			) {
				return errors.New("sessions.token_digestの長さCHECK制約が作成されていません")
			}

			var columns []string
			if err := tx.Raw(`
				SELECT column_name
				FROM information_schema.columns
				WHERE table_schema = current_schema()
				  AND table_name = 'sessions'
				ORDER BY ordinal_position
			`).Scan(&columns).Error; err != nil {
				return err
			}
			expectedColumns := []string{"id", "user_id", "token_digest", "expires_at", "created_at"}
			if strings.Join(columns, ",") != strings.Join(expectedColumns, ",") {
				return fmt.Errorf("sessions columns = %v, want %v", columns, expectedColumns)
			}
			return nil
		})
	})

	t.Run("session参照中のuser物理削除を拒否する", func(t *testing.T) {
		withMigratedDatabase(t, db, func(tx *gorm.DB) error {
			user, err := createSessionUser(tx, "delete")
			if err != nil {
				return err
			}
			digest := sha256.Sum256([]byte(fmt.Sprintf("delete-%d", time.Now().UnixNano())))
			if err := tx.Create(&model.Session{
				UserID:      user.ID,
				TokenDigest: digest[:],
				ExpiresAt:   time.Now().Add(time.Hour),
			}).Error; err != nil {
				return err
			}
			if err := tx.Delete(&user).Error; err == nil {
				return errors.New("session参照中のuser物理削除が拒否されませんでした")
			}
			return nil
		})
	})

	t.Run("user無効化後もsessionを保持する", func(t *testing.T) {
		withMigratedDatabase(t, db, func(tx *gorm.DB) error {
			user, err := createSessionUser(tx, "deactivate")
			if err != nil {
				return err
			}
			digest := sha256.Sum256([]byte(fmt.Sprintf("deactivate-%d", time.Now().UnixNano())))
			if err := tx.Create(&model.Session{
				UserID:      user.ID,
				TokenDigest: digest[:],
				ExpiresAt:   time.Now().Add(time.Hour),
			}).Error; err != nil {
				return err
			}
			if err := tx.Model(&user).Update("is_active", false).Error; err != nil {
				return err
			}
			var count int64
			if err := tx.Model(&model.Session{}).Where("user_id = ?", user.ID).Count(&count).Error; err != nil {
				return err
			}
			if count != 1 {
				return fmt.Errorf("無効化後のsession count = %d, want 1", count)
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

func createSessionUser(tx *gorm.DB, suffix string) (model.User, error) {
	var managerRole model.Role
	if err := tx.Where("name = ?", model.RoleNameManager).First(&managerRole).Error; err != nil {
		return model.User{}, err
	}
	user := model.User{
		Name:         "セッションテスト管理者",
		Email:        "session-" + suffix + "@example.invalid",
		PasswordHash: "test-password-hash",
		RoleID:       managerRole.ID,
	}
	if err := tx.Create(&user).Error; err != nil {
		return model.User{}, err
	}
	return user, nil
}
