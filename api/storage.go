package api

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

type Storage interface {
	CreateAccount(acc *Account) error
	DeleteAccount(id int) error
	UpdateAccountBalance(id int, balance int64) error
	GetAccountByID(id int) (*Account, error)
	GetAccountByEmail(email string) (*Account, error)
	GetAccountByAccountNumber(number int64) (*Account, error)
	GetAccounts() ([]*Account, error)
}

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore() (*PostgresStore, error) {
	connStr := "user=postgres dbname=gobankdb password=gobank sslmode=disable port=5433"
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) CreateAccount(acc *Account) error {
	query := `insert into account 
	(email, password, first_name, last_name, number, balance, created_at) values 
	($1, $2, $3, $4, $5, $6, $7)`
	
	resp, err := s.db.Query(
		query,
		acc.Email,
		acc.Password,
		acc.FirstName,
		acc.LastName,
		acc.Number,
		acc.Balance,
		acc.CreatedAt,
	)

	if err != nil {
		return err
	}
	log.Println(resp)
	return nil
}

func (s *PostgresStore) DeleteAccount(id int) error {

	query := `delete from account where id=$1`
	_, err := s.db.Exec(query, id)

	return err
}

func (s *PostgresStore) GetAccountByID(id int) (*Account, error) {
	query := `select id, email, first_name, last_name, number, balance, created_at from account where id=$1`
	row := s.db.QueryRow(query, id)

	account := &Account{}

	err := row.Scan(&account.ID, &account.Email, &account.FirstName, &account.LastName, &account.Number, &account.Balance, &account.CreatedAt)

	if err != nil {
		return nil, err
	}
	return account, nil
}

func (s *PostgresStore) GetAccountByAccountNumber(number int64) (*Account, error) {
	query := `select id, first_name, last_name, number, balance, created_at from account where number=$1`
	row := s.db.QueryRow(query, number)

	account := &Account{}

	err := row.Scan(&account.ID, &account.FirstName, &account.LastName, &account.Number, &account.Balance, &account.CreatedAt)

	if err != nil {
		return nil, err
	}
	return account, nil
}

func (s *PostgresStore) GetAccountByEmail(email string) (*Account, error) {
	query := `select id, email, password, first_name, last_name, number, balance, created_at from account where email=$1`
	row := s.db.QueryRow(query, email)

	account := &Account{}

	err := row.Scan(&account.ID, &account.Email, &account.Password, &account.FirstName, &account.LastName, &account.Number, &account.Balance, &account.CreatedAt)

	if err != nil {
		return nil, err
	}
	return account, nil
}


func (s *PostgresStore) GetAccounts() ([]*Account, error) {
	query := `select id, first_name, last_name, number, balance, created_at from account`

	rows, err := s.db.Query(query)

	if err != nil {
		return nil, err
	}
	accounts := []*Account{}
	for rows.Next() {
		account := &Account{}
		err := rows.Scan(&account.ID, &account.FirstName, &account.LastName, &account.Number, &account.Balance, &account.CreatedAt)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func (s *PostgresStore) Init() error {
	return s.createAccountTable()
}

func (s *PostgresStore) createAccountTable() error {
	query := `create table if not exists account(
		id serial primary key,
		email text,
		password text,
		first_name text,
		last_name text,
		number int,
		balance float,
		created_at timestamp
	)`
	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStore) UpdateAccountBalance(id int, balance int64) error {
	query := `update account set balance=balance + $1 where id=$2`
	_, err := s.db.Exec(query, balance, id)
	return err
}
