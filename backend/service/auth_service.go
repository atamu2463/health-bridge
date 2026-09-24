package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"time"

	"backend/model"
	"backend/repository"

	"golang.org/x/crypto/bcrypt"
)

const (
	sessionTokenSize = 32
	sessionLifetime  = 24 * time.Hour
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthenticated    = errors.New("unauthenticated")
	ErrInternal           = errors.New("authentication service unavailable")
)

// 有効なbcryptハッシュを使い、存在しないメールアドレスでもパスワード照合を行う。
var dummyPasswordHash = []byte("$2a$10$R9h/cIPz0gi.URNNX3kh2OPST9/PgBkqquzi.Ss7KIUgO2t0jWMUW")

type authRepository interface {
	FindUserByEmail(context.Context, string) (model.User, error)
	CreateSession(context.Context, model.Session) error
	FindSessionByTokenDigest(context.Context, []byte) (model.Session, error)
	DeleteSessionByTokenDigest(context.Context, []byte) error
}

type AuthenticatedUser struct {
	ID        uint
	Name      string
	Email     string
	Role      string
	ManagerID *uint
}

type LoginResult struct {
	User      AuthenticatedUser
	Token     string
	ExpiresAt time.Time
}

type AuthService struct {
	repository authRepository
	now        func() time.Time
	random     io.Reader
}

func NewAuthService(repository authRepository) *AuthService {
	return &AuthService{
		repository: repository,
		now:        time.Now,
		random:     rand.Reader,
	}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (LoginResult, error) {
	user, err := s.repository.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))
			return LoginResult{}, ErrInvalidCredentials
		}
		return LoginResult{}, ErrInternal
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil || !user.IsActive {
		return LoginResult{}, ErrInvalidCredentials
	}

	rawToken := make([]byte, sessionTokenSize)
	if _, err := io.ReadFull(s.random, rawToken); err != nil {
		return LoginResult{}, ErrInternal
	}

	createdAt := s.now()
	expiresAt := createdAt.Add(sessionLifetime)
	tokenDigest := sha256.Sum256(rawToken)
	if err := s.repository.CreateSession(ctx, model.Session{
		UserID:      user.ID,
		TokenDigest: tokenDigest[:],
		ExpiresAt:   expiresAt,
		CreatedAt:   createdAt,
	}); err != nil {
		return LoginResult{}, ErrInternal
	}

	return LoginResult{
		User:      authenticatedUser(user),
		Token:     base64.RawURLEncoding.EncodeToString(rawToken),
		ExpiresAt: expiresAt,
	}, nil
}

func (s *AuthService) Authenticate(ctx context.Context, token string) (AuthenticatedUser, error) {
	tokenDigest, err := digestToken(token)
	if err != nil {
		return AuthenticatedUser{}, ErrUnauthenticated
	}

	session, err := s.repository.FindSessionByTokenDigest(ctx, tokenDigest)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return AuthenticatedUser{}, ErrUnauthenticated
		}
		return AuthenticatedUser{}, ErrInternal
	}

	if !session.ExpiresAt.After(s.now()) || !session.User.IsActive {
		return AuthenticatedUser{}, ErrUnauthenticated
	}

	return authenticatedUser(session.User), nil
}

// Logoutは不正・削除済みトークンも成功扱いにし、セッションの存在を外部へ示さない。
func (s *AuthService) Logout(ctx context.Context, token string) error {
	tokenDigest, err := digestToken(token)
	if err != nil {
		return nil
	}

	if err := s.repository.DeleteSessionByTokenDigest(ctx, tokenDigest); err != nil {
		return ErrInternal
	}

	return nil
}

func digestToken(token string) ([]byte, error) {
	rawToken, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(rawToken) != sessionTokenSize || base64.RawURLEncoding.EncodeToString(rawToken) != token {
		return nil, ErrUnauthenticated
	}

	tokenDigest := sha256.Sum256(rawToken)
	return tokenDigest[:], nil
}

func authenticatedUser(user model.User) AuthenticatedUser {
	var managerID *uint
	if user.ManagerID != nil {
		id := *user.ManagerID
		managerID = &id
	}

	return AuthenticatedUser{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role.Name,
		ManagerID: managerID,
	}
}
