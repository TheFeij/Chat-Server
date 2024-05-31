package api

import (
	"Chat-Server/repository"
	"Chat-Server/util"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"net/http"
)

// signup route handler
func (s *server) signup(context *gin.Context) {
	var req SignupRequest

	if err := context.ShouldBindJSON(&req); err != nil {
		context.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		context.JSON(http.StatusInternalServerError, errorResponse(err))
	}

	newUser, err := s.repository.AddUser(&repository.User{
		Username: req.Username,
		Password: hashedPassword,
	})
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) {
			switch pgError.ConstraintName {
			case "users_pkey":
				context.JSON(http.StatusForbidden, errorResponse(err))
				return
			}
		}
		context.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	accessToken, accessTokenPayload, err := s.tokenMaker.CreateToken(
		newUser.Username,
		s.configs.AccessTokenDuration(),
	)
	if err != nil {
		context.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	refreshToken, refreshTokenPayload, err := s.tokenMaker.CreateToken(
		newUser.Username,
		s.configs.RefreshTokenDuration())
	if err != nil {
		context.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	res := SignupResponse{
		Username:              newUser.Username,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessTokenPayload.ExpiredAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshTokenPayload.ExpiredAt,
	}
	context.JSON(http.StatusOK, res)
}

// login route handler
func (s *server) login(context *gin.Context) {
	var req LoginRequest
	if err := context.ShouldBindJSON(&req); err != nil {
		context.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := s.repository.GetUser(req.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			context.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		context.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if err := util.CheckPassword(req.Password, user.Password); err != nil {
		context.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	accessToken, accessTokenPayload, err := s.tokenMaker.CreateToken(
		req.Username,
		s.configs.AccessTokenDuration(),
	)
	if err != nil {
		context.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	refreshToken, refreshTokenPayload, err := s.tokenMaker.CreateToken(
		req.Username,
		s.configs.RefreshTokenDuration())
	if err != nil {
		context.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	response := LoginResponse{
		Username:              user.Username,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessTokenPayload.ExpiredAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshTokenPayload.ExpiredAt,
	}
	context.JSON(http.StatusOK, response)
}

// errorResponse puts the error into a gin.H instance
func errorResponse(err error) gin.H {
	return gin.H{
		"error": err.Error(),
	}
}
