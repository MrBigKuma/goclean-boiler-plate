package controller

import (
	"errors"
	"github.com/gorilla/mux"
	"goclean/usecase"
	"net/http"
)

type UserCtrl interface {
	GetUser(w http.ResponseWriter, r *http.Request, uid string)
}

func NewUserCtrl(userUsecase usecase.UserUseCase) UserCtrl {
	return &userCtrlImpl{
		userUsecase: userUsecase,
	}
}

type userCtrlImpl struct {
	userUsecase usecase.UserUseCase
}

func (c *userCtrlImpl) GetUser(w http.ResponseWriter, r *http.Request, uid string) {
	// Get Uid in query
	vars := mux.Vars(r)
	userId, ok := vars["userId"]

	// Validate request data
	if !ok || userId == "" {
		ResponseError(w, http.StatusBadRequest, errors.New("missing userId"))
		return
	}

	// Call usecase layer to get user
	userEntity, err := c.userUsecase.GetUser(userId)
	if err != nil {
		ResponseError(w, http.StatusInternalServerError, err)
		return
	}
	if userEntity == nil {
		ResponseError(w, http.StatusNotFound, errors.New("User not found"))
		return
	}

	// Convert entity data to the new one that we will response to API
	userPresenter := NewUser(userEntity)

	ResponseOk(w, userPresenter)
}
