package auth

import "time"

type User struct {
	id           string
	clientId     string
	email        string
	mobile       string
	passwordHash string
	mpinHash     string
	hasMpin      bool
	isVerified   bool
	createdAt    time.Time
	updatedAt    time.Time
}

// Fields lowercase hain.

// Kyun?

// Because:

// 👉 Direct modification allowed nahi hona chahiye
// 👉 Domain state protected rehna chahiye

// We will expose getters later.

func NewUser(
	id string,
	clientId string,
	email Email,
	mobile Mobile,
	passwordHash string,
	mpinHash string,
	hasMpin bool,
) *User {

	now := time.Now()
	return &User{
		id:           id,
		clientId:     clientId,
		email:        string(email),
		mobile:       string(mobile),
		passwordHash: passwordHash,
		mpinHash:     mpinHash,
		isVerified:   false,
		hasMpin:      hasMpin,
		createdAt:    now,
		updatedAt:    now,
	}
}

// ⚠️ Note:

// ID generation domain ka kaam nahi hai
// UUID infrastructure/application generate karega
// Domain sirf accept karega

// ✅ Getters (Read-Only Access)
func (u *User) ID() string {
	return u.id
}

func (u *User) ClientID() string {
	return u.clientId
}

func (u *User) Email() Email {
	return Email(u.email)
}

func (u *User) Mobile() Mobile {
	return Mobile(u.mobile)
}

func (u *User) IsVerified() bool {
	return u.isVerified
}

func (u *User) IsHashMpin() bool {
	return u.hasMpin
}

func (u *User) CreatedAt() time.Time {
	return u.createdAt
}

func (u *User) UpdatedAt() time.Time {
	return u.updatedAt
}

func (u *User) MarkVerified() {
	u.isVerified = true
	u.updatedAt = time.Now()
}

func (u *User) MarkHashMpin() {
	u.hasMpin = true
	u.updatedAt = time.Now()
}

func (u *User) ChangePassword(newHash string) {
	u.passwordHash = newHash
	u.updatedAt = time.Now()
}

func (u *User) SetMPIN(hash string) {
	u.mpinHash = hash
	u.updatedAt = time.Now()
}

func (u *User) PasswordHash() string {
	return u.passwordHash
}

func (u *User) MPINHash() string {
	return u.mpinHash
}
