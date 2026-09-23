package main

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"backend/model"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const passwordHashCost = bcrypt.DefaultCost

var (
	errEmailAlreadyExists = errors.New("このメールアドレスは既に使用されています")
	errManagerNotFound    = errors.New("指定した有効なmanagerが見つかりません")
)

type createUserInput struct {
	Name         string
	Email        string
	Role         string
	ManagerEmail string
	Password     string
}

func createUser(db *gorm.DB, input createUserInput, bcryptCost int) (model.User, error) {
	normalized, err := validateCreateUserInput(input)
	if err != nil {
		return model.User{}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(normalized.Password), bcryptCost)
	if err != nil {
		return model.User{}, errors.New("パスワードを安全に保存できませんでした")
	}

	var createdUser model.User
	err = db.Transaction(func(tx *gorm.DB) error {
		var role model.Role
		if err := tx.Where("name = ?", normalized.Role).First(&role).Error; err != nil {
			return errors.New("rolesマスターデータを確認できませんでした")
		}

		var count int64
		if err := tx.Model(&model.User{}).
			Where("email = ?", normalized.Email).
			Count(&count).Error; err != nil {
			return errors.New("メールアドレスの重複を確認できませんでした")
		}
		if count > 0 {
			return errEmailAlreadyExists
		}

		createdUser = model.User{
			Name:         normalized.Name,
			Email:        normalized.Email,
			PasswordHash: string(passwordHash),
			RoleID:       role.ID,
		}

		if normalized.Role == model.RoleNameEmployee {
			var manager model.User
			if err := tx.Preload("Role").
				Where("email = ? AND is_active = ?", normalized.ManagerEmail, true).
				First(&manager).Error; err != nil || manager.Role.Name != model.RoleNameManager {
				return errManagerNotFound
			}
			createdUser.ManagerID = &manager.ID
		}

		if err := tx.Create(&createdUser).Error; err != nil {
			if isEmailUniqueViolation(err) {
				return errEmailAlreadyExists
			}
			return errors.New("ユーザーを作成できませんでした")
		}

		return nil
	})
	if err != nil {
		return model.User{}, err
	}

	return createdUser, nil
}

func validateCreateUserInput(input createUserInput) (createUserInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(input.Email)
	input.Role = strings.TrimSpace(input.Role)
	input.ManagerEmail = strings.TrimSpace(input.ManagerEmail)

	if input.Name == "" {
		return createUserInput{}, errors.New("名前を入力してください")
	}
	if input.Email == "" {
		return createUserInput{}, errors.New("メールアドレスを入力してください")
	}
	parsedEmail, err := mail.ParseAddress(input.Email)
	if err != nil || parsedEmail.Address != input.Email {
		return createUserInput{}, errors.New("メールアドレスの形式を確認してください")
	}
	if input.Role != model.RoleNameManager && input.Role != model.RoleNameEmployee {
		return createUserInput{}, fmt.Errorf("roleには%sまたは%sを指定してください", model.RoleNameManager, model.RoleNameEmployee)
	}
	if input.Password == "" {
		return createUserInput{}, errors.New("パスワードを入力してください")
	}
	if len([]byte(input.Password)) > 72 {
		return createUserInput{}, errors.New("パスワードは72バイト以内で入力してください")
	}

	if input.Role == model.RoleNameEmployee && input.ManagerEmail == "" {
		return createUserInput{}, errors.New("employeeにはmanagerのメールアドレスを指定してください")
	}
	if input.Role == model.RoleNameManager && input.ManagerEmail != "" {
		return createUserInput{}, errors.New("managerにはmanagerのメールアドレスを指定できません")
	}

	return input, nil
}

func isEmailUniqueViolation(err error) bool {
	var pgError *pgconn.PgError
	return errors.As(err, &pgError) &&
		pgError.Code == "23505" &&
		pgError.ConstraintName == "ux_users_email"
}
