package auth

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuthRepository defines the contract for user and refresh token persistence.
type AuthRepository interface {
	CreateUser(user *User) error
	FindUserByEmail(email string) (*User, error)
	FindUserByID(id uuid.UUID) (*User, error)
	SaveRefreshToken(token *RefreshToken) error
	FindRefreshTokenByHash(tokenHash string) (*RefreshToken, error)
	RevokeRefreshToken(tokenID uuid.UUID) error
	RevokeAllUserTokens(userID uuid.UUID) error
}

type postgresAuthRepository struct {
	db *gorm.DB
}

// NewAuthRepository creates a new PostgreSQL-backed auth repository.
func NewAuthRepository(db *gorm.DB) AuthRepository {
	return &postgresAuthRepository{db: db}
}

func (r *postgresAuthRepository) CreateUser(user *User) error {
	return r.db.Create(user).Error
}

func (r *postgresAuthRepository) FindUserByEmail(email string) (*User, error) {
	var user User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *postgresAuthRepository) FindUserByID(id uuid.UUID) (*User, error) {
	var user User
	err := r.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *postgresAuthRepository) SaveRefreshToken(token *RefreshToken) error {
	return r.db.Create(token).Error
}

func (r *postgresAuthRepository) FindRefreshTokenByHash(tokenHash string) (*RefreshToken, error) {
	var token RefreshToken
	err := r.db.Where("token_hash = ? AND revoked = false", tokenHash).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *postgresAuthRepository) RevokeRefreshToken(tokenID uuid.UUID) error {
	return r.db.Model(&RefreshToken{}).
		Where("id = ?", tokenID).
		Update("revoked", true).Error
}

func (r *postgresAuthRepository) RevokeAllUserTokens(userID uuid.UUID) error {
	return r.db.Model(&RefreshToken{}).
		Where("user_id = ? AND revoked = false", userID).
		Update("revoked", true).Error
}
