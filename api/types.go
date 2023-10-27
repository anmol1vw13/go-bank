package api

import (
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Email string `json:"email"`
	Password string `json:"password"`
}

type Account struct {
	ID        int       `json:"id"`
	Email string `json:"email"`
	Password string `json:"-"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Number    int64     `json:"number"`
	Balance   int64     `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
}

type TransferRequest struct {
	ToAccount int `json:"to_account"`
	Amount    int `json:"amount"`
}

type CreateAccountRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email string `json:"email"`
	Password string `json:"password"`
}

func NewAccount(req CreateAccountRequest) *Account {
	return &Account{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Number:    int64(rand.Intn(100000)),
		CreatedAt: time.Now().UTC(),
		Email: req.Email,
		Password: req.Password,
	}
}

func (account *Account) ValidatePassword(password string) error{
	return bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(password))
}
