package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ibra172/go-ffmpeg-pipeline/internal/apperr"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/httpresp"
)

type Handler struct {
	Service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		Service: service,
	}
}

// RegisterRequest contains the payload for creating a new user.
// swagger:model RegisterRequest
type RegisterRequest struct {
	Username string `json:"username" example:"alice"`
	Password string `json:"password" example:"password123"`
}

// RegisterResponse is returned after a successful registration.
// swagger:model RegisterResponse
type RegisterResponse struct {
	ID       string `json:"user_id"`
	Username string `json:"username"`
}

// LoginRequest contains the credentials for authentication.
// swagger:model LoginRequest
type LoginRequest struct {
	Username string `json:"username" example:"alice"`
	Password string `json:"password" example:"password123"`
}

// LoginResponse is returned after a successful login.
// swagger:model LoginResponse
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Register creates a new user account.
// @Summary Register new user
// @Description Creates a user account. Username: 3–30 ASCII letters, digits or underscores, starting with a letter; uniqueness is case-insensitive. Password: 8–72 bytes.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "User registration payload"
// @Success 201 {object} RegisterResponse "User registered successfully"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid JSON or registration parameters"
// @Failure 409 {object} httpresp.ErrorResponse "Username already exists"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var request RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httpresp.RespondError(
			ctx,
			w,
			fmt.Errorf("%w: %s", apperr.ErrInvalidArgument, err),
			"invalid request body",
		)
		return
	}

	user, err := h.Service.Register(ctx, request.Username, request.Password)
	if err != nil {
		publicMsg := "failed to register"

		var validationErr *validationError
		switch {
		case errors.As(err, &validationErr):
			publicMsg = validationErr.message
		case errors.Is(err, apperr.ErrConflict):
			publicMsg = "username is already taken"
		}

		httpresp.RespondError(ctx, w, err, publicMsg)
		return
	}

	httpresp.RespondJSON(w, http.StatusCreated, RegisterResponse{
		ID:       user.ID.String(),
		Username: user.Username,
	})
}

// Login authenticates a user and issues a session token.
// @Summary Login user
// @Description Authenticates an existing user and returns a valid bearer token for protected routes.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "User login payload"
// @Success 200 {object} LoginResponse "User logged in successfully"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body"
// @Failure 401 {object} httpresp.ErrorResponse "Invalid username or password"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var request LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httpresp.RespondError(
			ctx,
			w,
			fmt.Errorf("%w: %s", apperr.ErrInvalidArgument, err),
			"invalid request body",
		)
		return
	}

	session, err := h.Service.Login(ctx, request.Username, request.Password)
	if err != nil {
		httpresp.RespondError(ctx, w, err, "failed to log in")
		return
	}

	httpresp.RespondJSON(w, http.StatusOK, LoginResponse{
		Token:     session.Token,
		ExpiresAt: session.ExpiresAt,
	})
}
