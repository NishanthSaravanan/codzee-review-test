package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NishanthSaravanan/codzee-review-test/store"
)

func newTestServer() http.Handler {
	return NewAPI(store.NewTaskStore()).Handler()
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
	server := newTestServer()
	body := []byte(`{"title":"Assessment task","description":"Prepare a review"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
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
	getRec := httptest.NewRecorder()
	server.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d; body=%s", getRec.Code, http.StatusOK, getRec.Body.String())
	}
}

func TestCreateRejectsBlankTitle(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(`{"title":"   "}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestOversizedRequestBodyReturnsPayloadTooLarge(t *testing.T) {
	for _, test := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/tasks"},
		{method: http.MethodPut, path: "/tasks/1"},
	} {
		t.Run(test.method, func(t *testing.T) {
			body := strings.NewReader(`{"title":"` + strings.Repeat("a", 1<<20) + `"}`)
			req := httptest.NewRequest(test.method, test.path, body)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			newTestServer().ServeHTTP(rec, req)

			if rec.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusRequestEntityTooLarge, rec.Body.String())
			}
		})
	}
}

func TestMalformedJSONReturnsBadRequest(t *testing.T) {
	for _, test := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/tasks"},
		{method: http.MethodPut, path: "/tasks/1"},
	} {
		t.Run(test.method, func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.path, bytes.NewBufferString(`{"title":`))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			newTestServer().ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
		})
	}
}

func TestGetMissingTaskReturnsNotFound(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/tasks/999", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
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
