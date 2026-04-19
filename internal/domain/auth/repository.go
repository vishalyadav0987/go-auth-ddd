package auth

import "context"

type UserRepository interface {
	Save(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email Email) (*User, error)
	FindByMobile(ctx context.Context, mobile Mobile) (*User, error)
	FindByClientId(ctx context.Context, clientId string) (*User, error)
	FindById(ctx context.Context, userId string) (*User, error)
	UpdateMPIN(ctx context.Context, userId string, mpinHash string) error
	UpdatePassword(ctx context.Context, userID string, hashedPassword string) error
}

// 🧠 Why context?

// Because:

// DB call slow ho sakta hai
// Timeout cancel karna pad sakta hai
// Production safe code
