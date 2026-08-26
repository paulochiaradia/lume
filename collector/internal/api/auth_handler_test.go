package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHandleRefresh_Validations testa as camadas de segurança da borda
func TestHandleRefresh_Validations(t *testing.T) {
	s := &Server{}

	tests := []struct {
		name           string
		payload        interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Ataque 1: Body Totalmente Vazio",
			payload:        nil,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "body inválido",
		},
		{
			name:           "Ataque 2: JSON sem a chave refresh_token",
			payload:        map[string]string{"hacker_key": "123"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "refresh_token é obrigatório",
		},
		{
			name:           "Ataque 3: Refresh Token em Branco",
			payload:        map[string]string{"refresh_token": ""},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "refresh_token é obrigatório",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if tt.payload != nil {
				body, _ = json.Marshal(tt.payload)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(body))
			rr := httptest.NewRecorder()

			s.handleRefresh(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Status Code falhou! Esperado: %d, Recebido: %d", tt.expectedStatus, rr.Code)
			}

			var response map[string]string
			if err := json.NewDecoder(rr.Body).Decode(&response); err == nil {
				if response["error"] != tt.expectedError {
					t.Errorf("Mensagem falhou! Esperado: '%s', Recebido: '%s'", tt.expectedError, response["error"])
				}
			}
		})
	}
}

// TestHandleLogout_Validations garante que o Logout tem a mesma proteção
func TestHandleLogout_Validations(t *testing.T) {
	s := &Server{}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewBuffer([]byte(`{"refresh_token": ""}`)))
	rr := httptest.NewRecorder()

	s.handleLogout(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Esperava que Logout sem token desse status %d, mas deu %d", http.StatusBadRequest, rr.Code)
	}
}

// TestHandleLogin_Validations garante que campos obrigatórios sejam barrados na borda
func TestHandleLogin_Validations(t *testing.T) {
	s := &Server{}

	tests := []struct {
		name           string
		payload        interface{}
		expectedStatus int
	}{
		{
			name:           "Login Vazio",
			payload:        nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Login sem Email",
			payload:        map[string]string{"password": "123", "device_id": "pc"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Login sem Senha",
			payload:        map[string]string{"email": "teste@teste.com", "device_id": "pc"},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if tt.payload != nil {
				body, _ = json.Marshal(tt.payload)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
			rr := httptest.NewRecorder()

			s.handleLogin(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Status Code falhou no %s! Esperado: %d, Recebido: %d", tt.name, tt.expectedStatus, rr.Code)
			}
		})
	}
}

// TestProtectedRoutes_Unauthorized garante que as novas rotas estão com a porta trancada
func TestProtectedRoutes_Unauthorized(t *testing.T) {
	s := &Server{}

	// Como não estamos passando pelo middleware, o getClaims retornará nil.
	// Todos esses endpoints DEVERÃO retornar 401 Unauthorized imediatamente.
	routes := []struct {
		name    string
		method  string
		handler http.HandlerFunc
	}{
		{"Get Sessions", http.MethodGet, s.handleGetSessions},
		{"Revoke Session", http.MethodPost, s.handleRevokeSession},
		{"Global Logout", http.MethodPost, s.handleGlobalLogout},
		{"Logout Others", http.MethodPost, s.handleLogoutOthers},
	}

	for _, r := range routes {
		t.Run(r.name, func(t *testing.T) {
			req := httptest.NewRequest(r.method, "/dummy-url", nil)
			rr := httptest.NewRecorder()

			r.handler(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Errorf("Rota %s não protegeu o acesso! Esperado: 401, Recebido: %d", r.name, rr.Code)
			}
		})
	}
}
