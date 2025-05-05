package service

import (
	"context"
	"errors"
	"time"

	"user-service/internal/domain"
	"user-service/internal/repository"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	RegisterUser(ctx context.Context, user domain.User) (domain.User, error)
	AuthenticateUser(ctx context.Context, email, password string) (string, domain.User, error)
	GetUserProfile(ctx context.Context, id string) (domain.User, error)
	UpdateUser(ctx context.Context, user domain.User) error
}

type userService struct {
	userRepo    repository.UserRepository
	jwtSecret   string
	jwtDuration time.Duration
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func NewUserService(userRepo repository.UserRepository, jwtSecret string, jwtExpiryMinutes int) UserService {
	return &userService{
		userRepo:    userRepo,
		jwtSecret:   jwtSecret,
		jwtDuration: time.Duration(jwtExpiryMinutes) * time.Minute,
	}
}

func (s *userService) RegisterUser(ctx context.Context, user domain.User) (domain.User, error) {
	_, err := s.userRepo.GetByEmail(ctx, user.Email)
	if err == nil {
		return domain.User{}, errors.New("user with this email already exists")
	}

	return s.userRepo.Create(ctx, user)
}

func (s *userService) AuthenticateUser(ctx context.Context, email, password string) (string, domain.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", domain.User{}, errors.New("invalid email or password")
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", domain.User{}, errors.New("invalid email or password")
	}

	// Generate JWT token
	expirationTime := time.Now().Add(s.jwtDuration)
	claims := &Claims{
		UserID: user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", domain.User{}, err
	}

	// Clear password before returning
	user.Password = ""

	return tokenString, user, nil
}

func (s *userService) GetUserProfile(ctx context.Context, id string) (domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return domain.User{}, err
	}

	// Clear password before returning
	user.Password = ""

	return user, nil
}

func (s *userService) UpdateUser(ctx context.Context, user domain.User) error {
	return s.userRepo.Update(ctx, user)
}
