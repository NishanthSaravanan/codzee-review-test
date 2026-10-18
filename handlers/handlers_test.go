package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NishanthSaravanan/codzee-review-test/store"
	"github.com/golang-jwt/jwt/v5"
)

func newTestServer() http.Handler {
	return NewAPI(store.NewTaskStore()).Handler()
}

func newAuthenticatedTestServer(t *testing.T) (http.Handler, string) {
	t.Helper()
	t.Setenv("TASKFLOW_USERNAME", "reviewer")
	t.Setenv("TASKFLOW_PASSWORD", "test-password")
	server := NewAPI(store.NewTaskStore()).Handler()
	request := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"username":"reviewer","password":"test-password"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	var result loginResponse
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	return server, result.AccessToken
}

func TestHealth(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestCreateAndGetTask(t *testing.T) {
	server, token := newAuthenticatedTestServer(t)
	body := []byte(`{"title":"Assessment task","description":"Prepare a review"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)
	createRec := httptest.NewRecorder()

	server.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body=%s", createRec.Code, http.StatusCreated, createRec.Body.String())
	}

	var created store.Task
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created task: %v", err)
	}
	if created.ID != 1 {
		t.Fatalf("created ID = %d, want 1", created.ID)
	}
	if created.Status != store.StatusPending {
		t.Fatalf("created status = %q, want %q", created.Status, store.StatusPending)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/tasks/1", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getRec := httptest.NewRecorder()
	server.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d; body=%s", getRec.Code, http.StatusOK, getRec.Body.String())
	}
}

func TestCreateRejectsBlankTitle(t *testing.T) {
	server, token := newAuthenticatedTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(`{"title":"   "}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGetMissingTaskReturnsNotFound(t *testing.T) {
	server, token := newAuthenticatedTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/tasks/999", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestTaskEndpointsRequireAuthentication(t *testing.T) {
	server := NewAPI(store.NewTaskStore()).Handler()
	for _, test := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/tasks"},
		{method: http.MethodGet, path: "/tasks"},
		{method: http.MethodGet, path: "/tasks/1"},
		{method: http.MethodPut, path: "/tasks/1"},
		{method: http.MethodDelete, path: "/tasks/1"},
	} {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.path, nil)
			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
			}
		})
	}
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	t.Setenv("TASKFLOW_USERNAME", "reviewer")
	t.Setenv("TASKFLOW_PASSWORD", "test-password")
	server := NewAPI(store.NewTaskStore()).Handler()
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"username":"reviewer","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestTaskEndpointRejectsExpiredToken(t *testing.T) {
	server := NewAPI(store.NewTaskStore()).Handler()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
	})
	encoded, err := token.SignedString([]byte(jwtSigningSecret))
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+encoded)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestUpdateTask(t *testing.T) {
	taskStore := store.NewTaskStore()
	created, err := taskStore.Create(store.CreateTaskInput{Title: "Original"})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	updated, err := taskStore.Update(created.ID, store.UpdateTaskInput{
		Title:       "Updated",
		Description: "Updated description",
		Status:      store.StatusCompleted,
	})
	if err != nil {
		t.Fatalf("update task: %v", err)
	}
	if updated.Title != "Updated" || updated.Status != store.StatusCompleted {
		t.Fatalf("unexpected updated task: %+v", updated)
	}
}

func TestDeleteTask(t *testing.T) {
	taskStore := store.NewTaskStore()
	created, err := taskStore.Create(store.CreateTaskInput{Title: "Delete me"})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := taskStore.Delete(created.ID); err != nil {
		t.Fatalf("delete task: %v", err)
	}
	if _, err := taskStore.Get(created.ID); err != store.ErrTaskNotFound {
		t.Fatalf("get after delete error = %v, want %v", err, store.ErrTaskNotFound)
	}
}
