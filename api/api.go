package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

type APIServer struct {
	listenAddr string
	store      Storage
}

func NewAPIServer(listenAddr string, store Storage) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		store:      store,
	}
}

func (s *APIServer) Run() {
	router := mux.NewRouter()

	router.HandleFunc("/account", makeHTTPHandleFunc(s.handleAccount))
	router.HandleFunc("/account/{id}", withJWTAuth(makeHTTPHandleFunc(s.handleAccountByID)))
	router.HandleFunc("/transfer", withJWTAuth(makeHTTPHandleFunc(s.handleTransfer)))
	router.HandleFunc("/login", makeHTTPHandleFunc(s.handleLogin))
	log.Println("JSON API server running on port: ", s.listenAddr)
	err := http.ListenAndServe(fmt.Sprintf(":%s", s.listenAddr), router)
	if err != nil {
		log.Println("Encountered an error ", err)
	}

}

func (s *APIServer) handleLogin(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		return fmt.Errorf("Invalid Method")
	}
	
	loginRequest := LoginRequest{}
	if err := json.NewDecoder(r.Body).Decode(&loginRequest); err!=nil {
		return err
	}

	account, err := s.store.GetAccountByEmail(loginRequest.Email)
	if err != nil {
		return fmt.Errorf("Account with email %v doesnot exist", loginRequest.Email)
	}
	
	if account.ValidatePassword(loginRequest.Password) != nil {
		return fmt.Errorf("Invalid password")
	}

	token, err := createJWT(account)
	return WriteJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (s *APIServer) handleAccount(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return s.handleGetAccount(w, r)
	}
	if r.Method == "POST" {
		return s.handleCreateAccount(w, r)
	}

	return fmt.Errorf("Invalid Method %s", r.Method)
}

func (s *APIServer) handleAccountByID(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return s.handleGetAccountByID(w, r)
	}
	if r.Method == "DELETE" {
		return s.handleDeleteAccountByID(w, r)
	}

	return fmt.Errorf("Invalid Method %s", r.Method)
}

func (s *APIServer) handleGetAccount(w http.ResponseWriter, r *http.Request) error {

	accounts, err := s.store.GetAccounts()
	if err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, accounts)
}

func (s *APIServer) handleGetAccountByID(w http.ResponseWriter, r *http.Request) error {

	id, err := getId(r)
	if err != nil {
		return err
	}

	fmt.Println("Account no is", r.Context().Value("accountNumber"))

	account, err := s.store.GetAccountByID(id)
	if err != nil {
		return err
	}
	return WriteJSON(w, http.StatusOK, account)
}

func (s *APIServer) handleCreateAccount(w http.ResponseWriter, r *http.Request) error {
	createAccountReq := &CreateAccountRequest{}

	if err := json.NewDecoder(r.Body).Decode(createAccountReq); err != nil {
		return err
	}

	account := NewAccount(*createAccountReq)
	encryptedPassword, _ := bcrypt.GenerateFromPassword([]byte(account.Password), 1)
	account.Password = string(encryptedPassword)
	if err := s.store.CreateAccount(account); err != nil {
		return err
	}

	token, err := createJWT(account)

	if err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, map[string]any{"account": account, "token": token})
}

func (s *APIServer) handleDeleteAccountByID(w http.ResponseWriter, r *http.Request) error {
	id, err := getId(r)
	if err != nil {
		return err
	}
	err = s.store.DeleteAccount(id)
	if err != nil {
		return err
	}
	return WriteJSON(w, http.StatusOK, map[string]string{"message": "Account Has been deleted"})
}

func (s *APIServer) handleTransfer(w http.ResponseWriter, r *http.Request) error {

	if r.Method != "POST" {
		return fmt.Errorf("Invalid method %s", r.Method)
	}
	transferRequest := &TransferRequest{}

	if err := json.NewDecoder(r.Body).Decode(transferRequest); err != nil {
		return err
	}
	accNumVal := r.Context().Value("accountNumber")
	log.Println("Val is", accNumVal.(string))
	currentAccountNumber, err := strconv.Atoi(accNumVal.(string))
	if err != nil {
		return fmt.Errorf("Error converting %d", currentAccountNumber)
	}

	fromAccount, err := s.store.GetAccountByAccountNumber(int64(currentAccountNumber))
	if err != nil {
		return fmt.Errorf("Account number %v does not exist", fromAccount)
	}

	toAccount, err := s.store.GetAccountByAccountNumber(int64(transferRequest.ToAccount))
	if err != nil {
		return fmt.Errorf("Account number %v does not exist", transferRequest.ToAccount)
	}

	if fromAccount.Balance < int64(transferRequest.Amount) {
		return fmt.Errorf("You dont have sufficient balance")
	}

	err = s.store.UpdateAccountBalance(fromAccount.ID, -1*int64(transferRequest.Amount))

	if err != nil {
		return err
	}

	err = s.store.UpdateAccountBalance(toAccount.ID, int64(transferRequest.Amount))

	if err != nil {
		return err
	}
	defer r.Body.Close()
	return WriteJSON(w, http.StatusOK, map[string]string{"message": "Transfer is compeleted"})
}

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("content-type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

type apiFunc func(http.ResponseWriter, *http.Request) error

type ApiError struct {
	Error string `json:"error"`
}

func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			WriteJSON(w, http.StatusBadRequest, ApiError{Error: err.Error()})
		}
	}
}

func withJWTAuth(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		authToken := r.Header.Get("authorization")
		fmt.Println("Auth token", authToken)
		tokenStr := strings.Split(authToken, "Bearer ")[1]
		fmt.Println("This is the tokenStr", tokenStr)
		jwtToken, err := validateJWT(tokenStr)
		if err != nil {
			WriteJSON(w, http.StatusForbidden, ApiError{Error: err.Error()})
			return
		}
		claims := jwtToken.Claims.(jwt.MapClaims)
		log.Println(claims)
		accountNumber := claims["jti"]
		log.Println(accountNumber)
		ctx := context.WithValue(r.Context(), "accountNumber", accountNumber)
		handlerFunc(w, r.WithContext(ctx))
	}
}

func validateJWT(tokenString string) (*jwt.Token, error) {
	secret := jwtSecret
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}

func createJWT(account *Account) (string, error) {
	claims := &jwt.StandardClaims{
		ExpiresAt: time.Now().Unix() + 123123213,
		Id:        fmt.Sprint(account.Number),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

func getId(r *http.Request) (int, error) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		return 0, fmt.Errorf("Invalid id %s given", idStr)
	}
	return id, nil
}

const jwtSecret = "secret1706"