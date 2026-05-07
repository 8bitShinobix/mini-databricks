package service

import (
	"context"
	"fmt"
	"time"

	dbgen "github.com/8bitShinobix/mini-databricks/internal/db/generated"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	queries *dbgen.Queries
}

func NewUserService(queries *dbgen.Queries) *UserService {
	return &UserService{queries: queries}
}

func (s *UserService) Register(ctx context.Context, email, password string) (dbgen.User, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return dbgen.User{}, fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := s.queries.CreateUser(ctx, dbgen.CreateUserParams{
		Email:        email,
		PasswordHash: string(hashed),
		Role:         dbgen.UserRoleViewer,
	})
	if err != nil {
		return dbgen.User{}, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (s *UserService) Login(ctx context.Context, email, password, jwtSecret string) (string, error) {
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", fmt.Errorf("invalid credentials")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.String(),
		"role":    user.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	signed, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signed, nil
}
