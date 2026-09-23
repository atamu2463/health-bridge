package repository

import (
	"context"
	"errors"

	"backend/model"

	"gorm.io/gorm"
)

var (
	ErrNotFound    = errors.New("repository record not found")
	ErrUnavailable = errors.New("repository operation failed")
)

type AuthRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) FindUserByEmail(ctx context.Context, email string) (model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).
		Preload("Role").
		Where("email = ?", email).
		First(&user).Error; err != nil {
		return model.User{}, repositoryError(err)
	}

	return user, nil
}

func (r *AuthRepository) CreateSession(ctx context.Context, session model.Session) error {
	if err := r.db.WithContext(ctx).Create(&session).Error; err != nil {
		return ErrUnavailable
	}

	return nil
}

func (r *AuthRepository) FindSessionByTokenDigest(ctx context.Context, tokenDigest []byte) (model.Session, error) {
	var session model.Session
	if err := r.db.WithContext(ctx).
		Preload("User.Role").
		Where("token_digest = ?", tokenDigest).
		First(&session).Error; err != nil {
		return model.Session{}, repositoryError(err)
	}

	return session, nil
}

func (r *AuthRepository) DeleteSessionByTokenDigest(ctx context.Context, tokenDigest []byte) error {
	if err := r.db.WithContext(ctx).
		Where("token_digest = ?", tokenDigest).
		Delete(&model.Session{}).Error; err != nil {
		return ErrUnavailable
	}

	return nil
}

func repositoryError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}

	return ErrUnavailable
}
