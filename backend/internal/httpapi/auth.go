package httpapi

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/josbp0107/ia-energy-management/internal/config"
)

type auth struct {
	user  config.AuthConfig
	token string
}

func newAuth(user config.AuthConfig) (*auth, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil { // crypto/rand: aleatorio
		return nil, fmt.Errorf("generar token: %w", err)
	}
	return &auth{user: user, token: hex.EncodeToString(bytes)}, nil
}

// POST /auth/login  body: {"email": "...", "password": "..."}
func (a *auth) login(c *gin.Context) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "body JSON inválido", nil)
		return
	}

	emailOK := subtle.ConstantTimeCompare([]byte(strings.ToLower(body.Email)), []byte(strings.ToLower(a.user.Email))) == 1
	passwordOK := subtle.ConstantTimeCompare([]byte(body.Password), []byte(a.user.Password)) == 1
	if !emailOK || !passwordOK {
		respondError(c, http.StatusUnauthorized, "credenciales inválidas", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": a.token,
		"user":  gin.H{"email": a.user.Email, "name": a.user.Name},
	})
}

func (a *auth) requireToken(c *gin.Context) {
	token, found := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
	if !found || subtle.ConstantTimeCompare([]byte(token), []byte(a.token)) != 1 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no autenticado"})
		return
	}
	c.Next()
}
