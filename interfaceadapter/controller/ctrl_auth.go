package controller

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"goclean/interfaceadapter/repository"
	"goclean/usecase"
	"golang.org/x/crypto/sha3"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"
)

const AUTH_SALT = "xKuma-Stackx"
const PW_SALT_SIZE = 32
const DEFAULT_TOKEN_EXPIRED_MINUTE = 30 * 24 * 60 // 30 days
const RESET_TOKEN_EXPIRED_MINUTE = 30             // 30 minutes
const RESET_PASS_AUD = "resetPassAud"

type AuthCtrl interface {
	LoginByEmail(w http.ResponseWriter, r *http.Request)
	RegisterByMail(w http.ResponseWriter, r *http.Request)
}

type JwtAuth interface {
	CreateToken(uid, aud string, nExpiredDay int, signedKey string, now time.Time) (string, error)
	ParseToken(encryptedToken string, repoSignedKeyFunc func(uid, aud string) (string, error)) (string, error)
}

func NewAuthCtrl(resp Response, userUseCase usecase.UserUseCase, authRepo repository.AuthRepo, jwtAuth JwtAuth, mailManager MailManager) AuthCtrl {
	return &authCtrlImpl{
		userUseCase: userUseCase,
		response:    resp,
		authRepo:    authRepo,
		jwtAuth:     jwtAuth,
		mailManager: mailManager,
	}
}

type authCtrlImpl struct {
	userUseCase usecase.UserUseCase
	authRepo    repository.AuthRepo
	response    Response
	jwtAuth     JwtAuth
	mailManager MailManager
}

type registerByMailReq struct {
	email string `json:"email"`
	pass  string `json:"pass"`
}

type loginByEmailReq struct {
	email string `json:"email"`
	pass  string `json:"pass"`
	aud   string `json:"aud"`
}

func (c *authCtrlImpl) RegisterByMail(w http.ResponseWriter, r *http.Request) {
	// Read body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		c.response.Error(w, http.StatusBadRequest, err)
		return
	}
	req := registerByMailReq{}
	err = json.Unmarshal(body, &req)
	if err != nil {
		c.response.Error(w, http.StatusBadRequest, err)
		return
	}

	// Validate data
	// TODO: validate email format

	// TODO: validate password strength

	// TODO: Generate HashedPass & Salt
	salt := GenSalt()
	hashedPass, err := HashPass(req.pass, salt, AUTH_SALT)
	if err != nil {
		c.response.Error(w, http.StatusInternalServerError, err)
	}

	// Create a user
	userId, err := c.userUseCase.CreateUser()
	if err != nil {
		c.response.Error(w, http.StatusInternalServerError, err)
		return
	}

	// Create an auth data
	_, err = c.authRepo.CreateAuthByEmailAndHashPass(userId, req.email, hashedPass, salt)
	if err != nil {
		c.response.Error(w, http.StatusInternalServerError, err)
		return
	}

	// TODO: return body response here
	c.response.Ok(w, "")
}

func (c *authCtrlImpl) LoginByEmail(w http.ResponseWriter, r *http.Request) {
	// Read body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		c.response.Error(w, http.StatusBadRequest, err)
		return
	}
	req := loginByEmailReq{}
	err = json.Unmarshal(body, &req)
	if err != nil {
		c.response.Error(w, http.StatusBadRequest, err)
		return
	}

	// TODO: validation
	aud := req.aud
	if aud == "" {
		aud = "defaultAud"
	}

	auth, err := c.authRepo.GetByEmail(req.email)
	if err != nil {
		c.response.Error(w, http.StatusInternalServerError, err)
		return
	}
	if auth == nil {
		c.response.Error(w, http.StatusNotFound, errors.New("No Auth record"))
		return
	}

	// Validate pass
	if ValidatePass(req.pass, auth.HashedPass, auth.Salt, AUTH_SALT) {
		c.response.Error(w, http.StatusBadRequest, errors.New("Incorrect username & pass"))
		return
	}

	// Gen token and response
	signedKey := GenSalt()

	token, err := c.jwtAuth.CreateToken(auth.Uid, aud, DEFAULT_TOKEN_EXPIRED_MINUTE, signedKey, time.Now())
	if err != nil {
		c.response.Error(w, http.StatusInternalServerError, err)
		return
	}

	// Save signed key
	err = c.authRepo.SaveSignedKey(auth.Uid, aud, signedKey)
	if err != nil {
		c.response.Error(w, http.StatusInternalServerError, err)
		return
	}

	// TODO: Return token
	c.response.Ok(w, token)
}

func (c *authCtrlImpl) RequestResetPassword(w http.ResponseWriter, r *http.Request) {
	// Read body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		c.response.Error(w, http.StatusBadRequest, err)
		return
	}
	req := struct {
		email string `json:"email"`
	}{}
	err = json.Unmarshal(body, &req)
	if err != nil {
		c.response.Error(w, http.StatusBadRequest, err)
		return
	}

	// Get auth
	auth, err := c.authRepo.GetByEmail(req.email)
	if err != nil {
		c.response.Error(w, http.StatusInternalServerError, err)
		return
	}
	if auth == nil {
		// Response ok even user is not exist to prevent privacy leak
		c.response.Ok(w, "")
		return
	}

	// Gen token and response
	signedKey := GenSalt()
	resetToken, err := c.jwtAuth.CreateToken(auth.Uid, RESET_PASS_AUD, RESET_TOKEN_EXPIRED_MINUTE, signedKey, time.Now())
	if err != nil {
		c.response.Error(w, http.StatusInternalServerError, err)
		return
	}

	// Save signed key to database
	err = c.authRepo.SaveSignedKey(auth.Uid, RESET_PASS_AUD, signedKey)
	if err != nil {
		c.response.Error(w, http.StatusInternalServerError, err)
		return
	}

	// TODO: send mail to user
	c.mailManager.SendMail(Mail{
		ToList:  []string{auth.Email},
		Content: resetToken,
	})

	// TODO: response data later
	c.response.Ok(w, "")
}

// Create a salt with size equal PW_SALT_SIZE
// salt is response as a hex number
func GenSalt() string {
	b := make([]byte, PW_SALT_SIZE)
	rand.Read(b) // err is always nil
	return base64.StdEncoding.EncodeToString(b)
}

func HashPass(pass string, salt string, authSalt string) (string, error) {
	k, err := base64.StdEncoding.DecodeString(salt)
	if err != nil {
		return "", err
	}
	buf := []byte(pass)
	// A MAC with 32 bytes of output has 256-bit security strength -- if you use at least a 32-byte-long key.
	h := make([]byte, 32)
	d := sha3.NewShake256()
	// Write the key into the hash.
	combinedSalt := append(k, []byte(authSalt)...)
	d.Write(combinedSalt)
	// Now write the data.
	d.Write(buf)
	// Read 32 bytes of output from the hash into h.
	d.Read(h)

	return base64.StdEncoding.EncodeToString(h), nil
}

func ValidatePass(pass string, hashedPass string, salt string, authSalt string) bool {
	encoded, err := HashPass(pass, salt, authSalt)
	if err != nil {
		return false
	}
	return encoded == hashedPass
}
