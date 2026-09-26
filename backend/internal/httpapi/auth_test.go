package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/josbp0107/ia-energy-management/internal/config"
)

func testRouter(t *testing.T) (*gin.Engine, *auth) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	a, err := newAuth(config.AuthConfig{Email: "demo@bia.app", Password: "secreto", Name: "Demo"})
	if err != nil {
		t.Fatalf("newAuth: %v", err)
	}
	router := gin.New()
	router.Use(corsMiddleware("http://localhost:5173"))
	api := router.Group("/api/v1")
	api.POST("/auth/login", a.login)
	api.Group("", a.requireToken).GET("/meters", func(c *gin.Context) { c.JSON(http.StatusOK, []string{}) })
	return router, a
}

func do(router *gin.Engine, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestLogin(t *testing.T) {
	router, a := testRouter(t)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"credenciales correctas", `{"email":"demo@bia.app","password":"secreto"}`, http.StatusOK},
		{"email sin distinguir mayúsculas", `{"email":"DEMO@bia.app","password":"secreto"}`, http.StatusOK},
		{"contraseña incorrecta", `{"email":"demo@bia.app","password":"otra"}`, http.StatusUnauthorized},
		{"email incorrecto", `{"email":"x@bia.app","password":"secreto"}`, http.StatusUnauthorized},
		{"body inválido", `no es json`, http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(router, http.MethodPost, "/api/v1/auth/login", tc.body, nil)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, se esperaba %d (%s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus == http.StatusOK {
				var resp struct {
					Token string `json:"token"`
					User  struct {
						Name string `json:"name"`
					} `json:"user"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("respuesta no es JSON: %v", err)
				}
				if resp.Token != a.token || resp.User.Name != "Demo" {
					t.Errorf("respuesta inesperada: %s", rec.Body.String())
				}
			}
		})
	}
}

func TestRequireToken(t *testing.T) {
	router, a := testRouter(t)

	tests := []struct {
		name       string
		method     string
		headers    map[string]string
		wantStatus int
	}{
		{"sin token", http.MethodGet, nil, http.StatusUnauthorized},
		{"token incorrecto", http.MethodGet, map[string]string{"Authorization": "Bearer abc"}, http.StatusUnauthorized},
		{"sin prefijo Bearer", http.MethodGet, map[string]string{"Authorization": a.token}, http.StatusUnauthorized},
		{"token correcto", http.MethodGet, map[string]string{"Authorization": "Bearer " + a.token}, http.StatusOK},
		{"preflight CORS sin token", http.MethodOptions, map[string]string{"Origin": "http://localhost:5173"}, http.StatusNoContent},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(router, tc.method, "/api/v1/meters", "", tc.headers)
			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, se esperaba %d", rec.Code, tc.wantStatus)
			}
		})
	}
}

func TestTokensAreRandom(t *testing.T) {
	a1, err1 := newAuth(config.AuthConfig{})
	a2, err2 := newAuth(config.AuthConfig{})
	if err1 != nil || err2 != nil {
		t.Fatalf("newAuth: %v %v", err1, err2)
	}
	if a1.token == a2.token || len(a1.token) != 64 {
		t.Errorf("los tokens deben ser aleatorios de 64 caracteres hex: %q %q", a1.token, a2.token)
	}
}
