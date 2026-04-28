package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingyuins/octopus/internal/conf"
	"github.com/lingyuins/octopus/internal/db"
	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
	"github.com/lingyuins/octopus/internal/server/auth"
	"github.com/lingyuins/octopus/internal/server/middleware"
	"github.com/lingyuins/octopus/internal/server/resp"
)

func TestCreateUserThenLoginAsCreatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(t.Name()))
	if err := db.InitDB("sqlite", dsn, false); err != nil {
		t.Fatalf("init db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	conf.AppConfig.Auth.JWTSecret = "test-jwt-secret"

	if err := op.UserInit(); err != nil {
		t.Fatalf("user init: %v", err)
	}
	if err := op.UserBootstrapCreate("admin", "super-secret-123"); err != nil {
		t.Fatalf("bootstrap user: %v", err)
	}
	if err := op.InitCache(); err != nil {
		t.Fatalf("init cache: %v", err)
	}

	admin := op.UserGet()
	token, _, err := auth.GenerateJWTToken(60, admin.ID, admin.Role)
	if err != nil {
		t.Fatalf("generate admin token: %v", err)
	}

	engine := gin.New()
	engine.POST(
		"/api/v1/user/create",
		middleware.Auth(),
		middleware.RequireJSON(),
		middleware.RequirePermission(auth.PermUsersWrite),
		createUser,
	)
	engine.POST(
		"/api/v1/user/login",
		middleware.RequireJSON(),
		middleware.LoginRateLimit(),
		login,
	)

	createPayload := []byte(`{"username":"viewer","password":"viewer-secret-123","role":"viewer"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/user/create", bytes.NewReader(createPayload))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	engine.ServeHTTP(createRecorder, createReq)

	if createRecorder.Code != http.StatusOK {
		t.Fatalf("create user status = %d, want %d; body=%s", createRecorder.Code, http.StatusOK, createRecorder.Body.String())
	}

	loginPayload := []byte(`{"username":"viewer","password":"viewer-secret-123","expire":60}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/user/login", bytes.NewReader(loginPayload))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRecorder := httptest.NewRecorder()
	engine.ServeHTTP(loginRecorder, loginReq)

	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d; body=%s", loginRecorder.Code, http.StatusOK, loginRecorder.Body.String())
	}

	var response resp.ResponseStruct
	if err := json.Unmarshal(loginRecorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal login response: %v", err)
	}
	if response.Data == nil {
		t.Fatal("login response data is nil")
	}

	loginDataMap, ok := response.Data.(map[string]any)
	if !ok {
		t.Fatalf("login response data type = %T, want map[string]any", response.Data)
	}
	tokenValue, _ := loginDataMap["token"].(string)
	if strings.TrimSpace(tokenValue) == "" {
		t.Fatal("created user login token is empty")
	}

	valid, userID, role := auth.VerifyJWTToken(tokenValue)
	if !valid {
		t.Fatal("created user token is invalid")
	}
	if role != model.UserRoleViewer {
		t.Fatalf("created user token role = %q, want %q", role, model.UserRoleViewer)
	}

	createdUser, err := op.UserGetByUsername("viewer", context.Background())
	if err != nil {
		t.Fatalf("load created user: %v", err)
	}
	if userID != createdUser.ID {
		t.Fatalf("created user token id = %d, want %d", userID, createdUser.ID)
	}
}
