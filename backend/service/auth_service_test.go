package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"backend/model"
	"backend/repository"

	"golang.org/x/crypto/bcrypt"
)

var testNow = time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)

type fakeAuthRepository struct {
	users              map[string]model.User
	sessions           map[string]model.Session
	findUserErr        error
	createSessionErr   error
	findSessionErr     error
	deleteSessionErr   error
	lastCreatedSession *model.Session
	findSessionCalls   int
	deleteSessionCalls int
}

func newFakeAuthRepository() *fakeAuthRepository {
	return &fakeAuthRepository{
		users:    make(map[string]model.User),
		sessions: make(map[string]model.Session),
	}
}

func (r *fakeAuthRepository) FindUserByEmail(_ context.Context, email string) (model.User, error) {
	if r.findUserErr != nil {
		return model.User{}, r.findUserErr
	}
	user, ok := r.users[email]
	if !ok {
		return model.User{}, repository.ErrNotFound
	}
	return user, nil
}

func (r *fakeAuthRepository) CreateSession(_ context.Context, session model.Session) error {
	if r.createSessionErr != nil {
		return r.createSessionErr
	}
	session.TokenDigest = append([]byte(nil), session.TokenDigest...)
	for _, user := range r.users {
		if user.ID == session.UserID {
			session.User = user
			break
		}
	}
	r.sessions[string(session.TokenDigest)] = session
	stored := session
	r.lastCreatedSession = &stored
	return nil
}

func (r *fakeAuthRepository) FindSessionByTokenDigest(_ context.Context, tokenDigest []byte) (model.Session, error) {
	r.findSessionCalls++
	if r.findSessionErr != nil {
		return model.Session{}, r.findSessionErr
	}
	session, ok := r.sessions[string(tokenDigest)]
	if !ok {
		return model.Session{}, repository.ErrNotFound
	}
	return session, nil
}

func (r *fakeAuthRepository) DeleteSessionByTokenDigest(_ context.Context, tokenDigest []byte) error {
	r.deleteSessionCalls++
	if r.deleteSessionErr != nil {
		return r.deleteSessionErr
	}
	delete(r.sessions, string(tokenDigest))
	return nil
}

func TestLoginSuccess(t *testing.T) {
	password := "correct-password"
	repository := newFakeAuthRepository()
	repository.users["manager@example.invalid"] = testUser(t, password, true)
	randomToken := bytes.Repeat([]byte{0xab}, sessionTokenSize)
	service := testAuthService(repository, randomToken)

	result, err := service.Login(context.Background(), "manager@example.invalid", password)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.User.ID != 1 || result.User.Role != model.RoleNameManager {
		t.Fatal("Login() returned an unexpected authenticated user")
	}
	if result.Token != base64.RawURLEncoding.EncodeToString(randomToken) {
		t.Fatal("Login() token does not match the generated random bytes")
	}
	if strings.ContainsAny(result.Token, "+/=") {
		t.Fatal("Login() token is not unpadded Base64 URL format")
	}
	decodedToken, err := base64.RawURLEncoding.DecodeString(result.Token)
	if err != nil {
		t.Fatalf("Login() token cannot be decoded: %v", err)
	}
	if len(decodedToken) != sessionTokenSize {
		t.Fatalf("decoded token length = %d, want %d", len(decodedToken), sessionTokenSize)
	}

	stored := repository.lastCreatedSession
	if stored == nil {
		t.Fatal("session was not created")
	}
	wantDigest := sha256.Sum256(randomToken)
	if !bytes.Equal(stored.TokenDigest, wantDigest[:]) {
		t.Fatal("stored digest does not match the generated token")
	}
	if len(stored.TokenDigest) != model.SessionTokenDigestSize {
		t.Fatalf("stored digest length = %d, want %d", len(stored.TokenDigest), model.SessionTokenDigestSize)
	}
	if bytes.Equal(stored.TokenDigest, []byte(result.Token)) || bytes.Contains(stored.TokenDigest, []byte(result.Token)) {
		t.Fatal("client token was stored without hashing")
	}
	if !stored.CreatedAt.Equal(testNow) {
		t.Fatalf("CreatedAt = %v, want %v", stored.CreatedAt, testNow)
	}
	if !stored.ExpiresAt.Equal(testNow.Add(24*time.Hour)) || !result.ExpiresAt.Equal(stored.ExpiresAt) {
		t.Fatalf("ExpiresAt = %v, want %v", stored.ExpiresAt, testNow.Add(24*time.Hour))
	}
}

func TestDummyPasswordHashIsValid(t *testing.T) {
	if _, err := bcrypt.Cost(dummyPasswordHash); err != nil {
		t.Fatalf("dummyPasswordHash is not a valid bcrypt hash: %v", err)
	}
}

func TestLoginUsesBcryptPasswordComparison(t *testing.T) {
	repository := newFakeAuthRepository()
	user := testUser(t, "correct-password", true)
	user.PasswordHash = "correct-password"
	repository.users[user.Email] = user

	_, err := testAuthService(repository, bytes.Repeat([]byte{1}, sessionTokenSize)).Login(
		context.Background(),
		user.Email,
		"correct-password",
	)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
	if repository.lastCreatedSession != nil {
		t.Fatal("session was created for a plaintext password_hash")
	}
}

func TestLoginRejectsInvalidCredentialsAndInactiveUser(t *testing.T) {
	activeUser := testUser(t, "correct-password", true)
	inactiveUser := testUser(t, "correct-password", false)
	inactiveUser.ID = 2
	inactiveUser.Email = "inactive@example.invalid"

	tests := []struct {
		name     string
		email    string
		password string
	}{
		{name: "存在しないメールアドレス", email: "missing@example.invalid", password: "correct-password"},
		{name: "パスワード不一致", email: activeUser.Email, password: "wrong-password"},
		{name: "無効化ユーザー", email: inactiveUser.Email, password: "correct-password"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := newFakeAuthRepository()
			repository.users[activeUser.Email] = activeUser
			repository.users[inactiveUser.Email] = inactiveUser
			_, err := testAuthService(repository, bytes.Repeat([]byte{1}, sessionTokenSize)).Login(
				context.Background(),
				tt.email,
				tt.password,
			)
			if err != ErrInvalidCredentials {
				t.Fatalf("Login() error = %v, want the shared ErrInvalidCredentials", err)
			}
			if repository.lastCreatedSession != nil {
				t.Fatal("session was created for rejected credentials")
			}
		})
	}
}

func TestAuthenticateSuccess(t *testing.T) {
	repository := newFakeAuthRepository()
	user := testUser(t, "password", true)
	token := testToken(0x11)
	repository.sessions[digestKey(t, token)] = model.Session{
		UserID:    user.ID,
		User:      user,
		ExpiresAt: testNow.Add(time.Hour),
	}

	got, err := testAuthService(repository, nil).Authenticate(context.Background(), token)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if got.ID != user.ID || got.Role != model.RoleNameManager {
		t.Fatal("Authenticate() returned an unexpected authenticated user")
	}
}

func TestAuthenticateRejectsInvalidTokenBeforeDatabaseLookup(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{name: "不正なBase64", token: "not+base64"},
		{name: "長さ不足", token: base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{1}, sessionTokenSize-1))},
		{name: "長さ超過", token: base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{1}, sessionTokenSize+1))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := newFakeAuthRepository()
			_, err := testAuthService(repository, nil).Authenticate(context.Background(), tt.token)
			if !errors.Is(err, ErrUnauthenticated) {
				t.Fatalf("Authenticate() error = %v, want ErrUnauthenticated", err)
			}
			if repository.findSessionCalls != 0 {
				t.Fatalf("FindSessionByTokenDigest() calls = %d, want 0", repository.findSessionCalls)
			}
		})
	}
}

func TestAuthenticateRejectsUnavailableSession(t *testing.T) {
	activeUser := testUser(t, "password", true)
	inactiveUser := testUser(t, "password", false)
	inactiveUser.ID = 2

	tests := []struct {
		name       string
		addSession bool
		user       model.User
		expiresAt  time.Time
	}{
		{name: "存在しないセッション"},
		{name: "期限切れセッション", addSession: true, user: activeUser, expiresAt: testNow},
		{name: "無効化済みユーザー", addSession: true, user: inactiveUser, expiresAt: testNow.Add(time.Hour)},
	}

	for index, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := newFakeAuthRepository()
			token := testToken(byte(index + 1))
			if tt.addSession {
				repository.sessions[digestKey(t, token)] = model.Session{
					UserID:    tt.user.ID,
					User:      tt.user,
					ExpiresAt: tt.expiresAt,
				}
			}

			_, err := testAuthService(repository, nil).Authenticate(context.Background(), token)
			if !errors.Is(err, ErrUnauthenticated) {
				t.Fatalf("Authenticate() error = %v, want ErrUnauthenticated", err)
			}
		})
	}
}

func TestLogoutRevokesSessionAndIsIdempotent(t *testing.T) {
	repository := newFakeAuthRepository()
	user := testUser(t, "correct-password", true)
	repository.users[user.Email] = user
	service := testAuthService(repository, bytes.Repeat([]byte{0x42}, sessionTokenSize))

	login, err := service.Login(context.Background(), user.Email, "correct-password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if _, err := service.Authenticate(context.Background(), login.Token); err != nil {
		t.Fatalf("Authenticate() before logout error = %v", err)
	}
	if err := service.Logout(context.Background(), login.Token); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, err := service.Authenticate(context.Background(), login.Token); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("Authenticate() after logout error = %v, want ErrUnauthenticated", err)
	}
	if err := service.Logout(context.Background(), login.Token); err != nil {
		t.Fatalf("second Logout() error = %v", err)
	}
	if err := service.Logout(context.Background(), "invalid+token"); err != nil {
		t.Fatalf("Logout() with invalid token error = %v", err)
	}
}

func TestAuthenticationErrorsDoNotExposeSecrets(t *testing.T) {
	secret := "postgresql://user:database-password@example.invalid/db"
	password := "user-password-secret"
	token := testToken(0x33)
	user := testUser(t, password, true)

	tests := []struct {
		name string
		run  func(*fakeAuthRepository) error
	}{
		{
			name: "user lookup failure",
			run: func(repository *fakeAuthRepository) error {
				repository.findUserErr = errors.New(secret)
				_, err := testAuthService(repository, nil).Login(context.Background(), user.Email, password)
				return err
			},
		},
		{
			name: "session creation failure",
			run: func(repository *fakeAuthRepository) error {
				repository.users[user.Email] = user
				repository.createSessionErr = errors.New(secret)
				_, err := testAuthService(repository, bytes.Repeat([]byte{1}, sessionTokenSize)).Login(context.Background(), user.Email, password)
				return err
			},
		},
		{
			name: "random source failure",
			run: func(repository *fakeAuthRepository) error {
				repository.users[user.Email] = user
				service := testAuthService(repository, nil)
				service.random = errorReader{err: errors.New(secret)}
				_, err := service.Login(context.Background(), user.Email, password)
				return err
			},
		},
		{
			name: "session lookup failure",
			run: func(repository *fakeAuthRepository) error {
				repository.findSessionErr = errors.New(secret)
				_, err := testAuthService(repository, nil).Authenticate(context.Background(), token)
				return err
			},
		},
		{
			name: "session deletion failure",
			run: func(repository *fakeAuthRepository) error {
				repository.deleteSessionErr = errors.New(secret)
				return testAuthService(repository, nil).Logout(context.Background(), token)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(newFakeAuthRepository())
			if !errors.Is(err, ErrInternal) {
				t.Fatalf("error = %v, want ErrInternal", err)
			}
			for _, sensitiveValue := range []string{secret, password, token} {
				if strings.Contains(err.Error(), sensitiveValue) {
					t.Fatalf("error contains sensitive value: %q", err)
				}
			}
		})
	}
}

func testAuthService(repository authRepository, randomToken []byte) *AuthService {
	randomSource := io.Reader(bytes.NewReader(randomToken))
	if randomToken == nil {
		randomSource = bytes.NewReader(nil)
	}
	return &AuthService{
		repository: repository,
		now:        func() time.Time { return testNow },
		random:     randomSource,
	}
}

func testUser(t *testing.T, password string, isActive bool) model.User {
	t.Helper()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt.GenerateFromPassword() error = %v", err)
	}
	return model.User{
		ID:           1,
		Name:         "テスト管理者",
		Email:        "manager@example.invalid",
		PasswordHash: string(passwordHash),
		RoleID:       1,
		Role:         model.Role{ID: 1, Name: model.RoleNameManager},
		IsActive:     isActive,
	}
}

func testToken(value byte) string {
	return base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{value}, sessionTokenSize))
}

func digestKey(t *testing.T, token string) string {
	t.Helper()
	digest, err := digestToken(token)
	if err != nil {
		t.Fatalf("digestToken() error = %v", err)
	}
	return string(digest)
}

type errorReader struct {
	err error
}

func (r errorReader) Read(_ []byte) (int, error) {
	return 0, r.err
}
