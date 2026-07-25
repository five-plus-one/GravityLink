package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gravitylink/backend/internal/config"
	"gravitylink/backend/internal/model"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountUnavailable = errors.New("account unavailable")
)

const localSessionTTL = 12 * time.Hour

type AuthService struct {
	db *gorm.DB
}

type Identity struct {
	Subject  string
	Username string
	Email    string
}

func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{db: db}
}

func (s *AuthService) Installed() (bool, error) {
	var state model.InstallationState
	err := s.db.First(&state, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return state.Installed && state.OwnerUserID != nil, err
}

func HashPassword(password string) (string, error) {
	if len(password) < 10 {
		return "", errors.New("password must contain at least 10 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func (s *AuthService) CreateBootstrapOwner(cfg config.Config) (model.User, error) {
	var owner model.User
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var state model.InstallationState
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&state, 1).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			state = model.InstallationState{ID: 1, SchemaVersion: 1}
			if err := tx.Create(&state).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		if state.Installed && state.OwnerUserID != nil {
			return tx.First(&owner, *state.OwnerUserID).Error
		}
		if !cfg.BootstrapVerified {
			return errors.New("administrator identity has not been verified")
		}

		switch cfg.AuthMode {
		case model.AuthSourceLocal:
			err = tx.Where("auth_source = ? AND username = ?", model.AuthSourceLocal, cfg.BootstrapUsername).First(&owner).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				email := authOptionalString(cfg.BootstrapEmail)
				passwordHash := cfg.BootstrapPassword
				owner = model.User{
					AuthSource: model.AuthSourceLocal, Username: cfg.BootstrapUsername, Email: email,
					PasswordHash: &passwordHash, Role: model.UserRoleSuperAdmin, Status: model.StatusActive,
				}
				if err := tx.Create(&owner).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
		default:
			subject := cfg.BootstrapSubject
			err = tx.Where("sso_id = ?", subject).First(&owner).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				owner = model.User{
					AuthSource: model.AuthSourceLogto, SSOID: &subject, Username: cfg.BootstrapUsername,
					Email: authOptionalString(cfg.BootstrapEmail), Role: model.UserRoleSuperAdmin, Status: model.StatusActive,
				}
				if err := tx.Create(&owner).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
		}

		if err := tx.Model(&owner).Updates(map[string]any{
			"role": model.UserRoleSuperAdmin, "status": model.StatusActive,
		}).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		state.Installed = true
		state.OwnerUserID = &owner.ID
		state.SchemaVersion = 1
		state.InstalledAt = &now
		return tx.Save(&state).Error
	})
	return owner, err
}

func (s *AuthService) ProvisionLogto(identity Identity) (model.User, error) {
	var user model.User
	err := s.db.Where("sso_id = ?", identity.Subject).First(&user).Error
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return user, err
	}
	subject := identity.Subject
	username := firstValue(identity.Username, identity.Email, "logto-"+shortSubject(subject))
	user = model.User{
		AuthSource: model.AuthSourceLogto, SSOID: &subject, Username: username,
		Email: authOptionalString(identity.Email), Role: model.UserRoleUser, Status: model.StatusPending,
	}
	err = s.db.Create(&user).Error
	if err != nil {
		if lookupErr := s.db.Where("sso_id = ?", identity.Subject).First(&user).Error; lookupErr == nil {
			return user, nil
		}
	}
	return user, err
}

func (s *AuthService) LoginLocal(username string, password string) (model.User, string, error) {
	var user model.User
	if err := s.db.Where("auth_source = ? AND username = ?", model.AuthSourceLocal, strings.TrimSpace(username)).First(&user).Error; err != nil {
		return user, "", ErrInvalidCredentials
	}
	if user.PasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)) != nil {
		return user, "", ErrInvalidCredentials
	}
	if user.Status != model.StatusActive {
		return user, "", ErrAccountUnavailable
	}
	token, hash, err := newSessionToken()
	if err != nil {
		return user, "", err
	}
	now := time.Now().UTC()
	session := model.AuthSession{UserID: user.ID, TokenHash: hash, ExpiresAt: now.Add(localSessionTTL), LastSeenAt: &now}
	if err := s.db.Create(&session).Error; err != nil {
		return user, "", err
	}
	_ = s.db.Model(&user).Update("last_login_at", now).Error
	return user, token, nil
}

func (s *AuthService) VerifyLocalPassword(userID uint64, password string) error {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return ErrInvalidCredentials
	}
	if user.AuthSource != model.AuthSourceLocal || user.PasswordHash == nil {
		return ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)) != nil {
		return ErrInvalidCredentials
	}
	return nil
}

func (s *AuthService) ResolveSession(token string) (model.User, error) {
	var user model.User
	hash := hashToken(token)
	err := s.db.Table("users").
		Joins("JOIN auth_sessions ON auth_sessions.user_id = users.id").
		Where("auth_sessions.token_hash = ? AND auth_sessions.expires_at > ? AND users.status = ?", hash, time.Now().UTC(), model.StatusActive).
		Where("users.deleted_at IS NULL").
		First(&user).Error
	if err != nil {
		return user, ErrInvalidCredentials
	}
	return user, nil
}

func (s *AuthService) RevokeSession(token string) error {
	if token == "" {
		return nil
	}
	return s.db.Where("token_hash = ?", hashToken(token)).Delete(&model.AuthSession{}).Error
}

func (s *AuthService) RevokeAllSessions() error {
	return s.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.AuthSession{}).Error
}

func newSessionToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(raw)
	return token, hashToken(token), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func authOptionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func firstValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func shortSubject(subject string) string {
	if len(subject) <= 12 {
		return subject
	}
	return subject[:12]
}
