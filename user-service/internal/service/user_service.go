package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"user-service/internal/cache"
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
	cache       cache.Cache
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func NewUserService(userRepo repository.UserRepository, jwtSecret string, jwtExpiryMinutes int, cache cache.Cache) UserService {
	return &userService{
		userRepo:    userRepo,
		jwtSecret:   jwtSecret,
		jwtDuration: time.Duration(jwtExpiryMinutes) * time.Minute,
		cache:       cache,
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
	// Try to get from cache first
	cacheKey := fmt.Sprintf("user:profile:%s", id)
	var cachedUser domain.User

	err := s.cache.Get(ctx, cacheKey, &cachedUser)
	if err == nil {
		log.Printf("Cache hit for user profile ID: %s", id)
		return cachedUser, nil
	}

	if err != cache.ErrCacheMiss {
		log.Printf("Cache error for user profile ID %s: %v", id, err)
	}

	// If not in cache, get from database
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return domain.User{}, err
	}

	// Clear password before caching
	user.Password = ""

	// Store in cache with 15-minute TTL
	if err := s.cache.Set(ctx, cacheKey, user, 15*time.Minute); err != nil {
		log.Printf("Failed to cache user profile ID %s: %v", id, err)
	}

	return user, nil
}

func (s *userService) UpdateUser(ctx context.Context, user domain.User) error {
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("user:profile:%s", user.ID)
	if err := s.cache.Delete(ctx, cacheKey); err != nil {
		log.Printf("Failed to invalidate cache for user profile ID %s: %v", user.ID, err)
	}

	return nil
}
