package auth

import (
	"encoding/json"
	"net/http"

	"github.com/sperax/flight-price-service/internal/httpx"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type Handler struct {
	username          string
	password          string
	jwtSecret         string
	expirationMinutes int
}

func NewHandler(username, password, jwtSecret string, expirationMinutes int) *Handler {
	return &Handler{
		username:          username,
		password:          password,
		jwtSecret:         jwtSecret,
		expirationMinutes: expirationMinutes,
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.BadRequest(w, "invalid request body")
		return
	}

	if req.Username != h.username || req.Password != h.password {
		httpx.Unauthorized(w, "invalid credentials")
		return
	}

	token, err := GenerateToken(h.jwtSecret, h.expirationMinutes)
	if err != nil {
		httpx.InternalError(w)
		return
	}

	httpx.JSON(w, http.StatusOK, loginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   h.expirationMinutes * 60,
	})
}
