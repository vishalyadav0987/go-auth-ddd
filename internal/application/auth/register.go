package authapp

import (
	"context"

	domain "github.com/vishalyadav0987/authentication/internal/domain/auth"
)

type PasswordHasher interface {
	Hash(password string) (string, error)
	ComparePassword(hashedPassword, password string) error
}

type IDGenerator interface {
	Generate() string
}

type RegisterUsecase struct {
	repo   domain.UserRepository
	hasher PasswordHasher
	idGen  IDGenerator
}

func NewRegisterUsecase(
	repo domain.UserRepository,
	hasher PasswordHasher,
	idGen IDGenerator,
) *RegisterUsecase {
	return &RegisterUsecase{
		repo:   repo,
		hasher: hasher,
		idGen:  idGen,
	}
}

// Application layer input structure:
type RegisterRequest struct {
	Email    string
	Password string
	Mobile   string
}

func (uc *RegisterUsecase) Execute(
	ctx context.Context,
	req RegisterRequest,
) (*domain.User, error) {
	// 1. value Object
	email, err := domain.NewEmail(req.Email)
	if err != nil {
		return nil, err
	}

	mobile, err := domain.NewMobile(req.Mobile)
	if err != nil {
		return nil, err
	}

	password, err := domain.NewPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// 2. Check existing user
	existingUser, _ := uc.repo.FindByEmail(ctx, domain.Email(req.Email))
	if existingUser != nil {
		return nil, domain.ErrUserAlreadyExists
	}

	// 3. hash the password
	hashedPassword, err := uc.hasher.Hash(string(password))
	if err != nil {
		return nil, err
	}

	// 4. Generate IDs
	id := uc.idGen.Generate()
	clientID := uc.idGen.Generate()

	// 5. Save user using interface
	user := domain.NewUser(
		id,
		clientID,
		email,
		mobile,
		hashedPassword,
		"",    //mpinHash
		false, //hasMpin
	)

	// 6. Save
	err = uc.repo.Save(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil

}
