package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestGetMissingTaskReturnsNotFound(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/tasks/999", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestSearchTasksFiltersAndPaginates(t *testing.T) {
	taskStore := store.NewTaskStore()
	first, err := taskStore.Create(store.CreateTaskInput{Title: "Write report", Description: "Prepare findings"})
	if err != nil {
		t.Fatalf("create first task: %v", err)
	}
	second, err := taskStore.Create(store.CreateTaskInput{Title: "Review report", Description: "Check results"})
	if err != nil {
		t.Fatalf("create second task: %v", err)
	}
	if _, err := taskStore.Update(second.ID, store.UpdateTaskInput{
		Title:       second.Title,
		Description: second.Description,
		Status:      store.StatusCompleted,
	}); err != nil {
		t.Fatalf("complete second task: %v", err)
	}
	third, err := taskStore.Create(store.CreateTaskInput{Title: "Draft report", Description: "Summarize findings"})
	if err != nil {
		t.Fatalf("create third task: %v", err)
	}
	server := NewAPI(taskStore).Handler()

	req := httptest.NewRequest(http.MethodGet, "/tasks/search?q=REPORT&status=pending&page=1&page_size=1", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var result store.TaskSearchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode search result: %v", err)
	}
	if result.Total != 2 || result.Page != 1 || result.PageSize != 1 || len(result.Items) != 1 {
		t.Fatalf("unexpected search result: %+v", result)
	}
	if result.Items[0].ID != first.ID {
		t.Fatalf("result task ID = %d, want %d", result.Items[0].ID, first.ID)
	}

	req = httptest.NewRequest(http.MethodGet, "/tasks/search?q=report&status=pending&page=2&page_size=1", nil)
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode second page: %v", err)
	}
	if rec.Code != http.StatusOK || result.Total != 2 || result.Page != 2 || len(result.Items) != 1 || result.Items[0].ID != third.ID {
		t.Fatalf("unexpected second page: status=%d result=%+v", rec.Code, result)
	}
}

func TestSearchTasksReturnsEmptyItemsForOutOfRangePage(t *testing.T) {
	taskStore := store.NewTaskStore()
	if _, err := taskStore.Create(store.CreateTaskInput{Title: "One task"}); err != nil {
		t.Fatalf("create task: %v", err)
	}
	server := NewAPI(taskStore).Handler()
	req := httptest.NewRequest(http.MethodGet, "/tasks/search?page=2&page_size=1", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var result store.TaskSearchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode search result: %v", err)
	}
	if result.Total != 1 || result.Items == nil || len(result.Items) != 0 {
		t.Fatalf("unexpected out-of-range result: %+v", result)
	}
}

func TestSearchTasksRejectsInvalidParameters(t *testing.T) {
	server := newTestServer()
	for _, target := range []string{
		"/tasks/search?status=unknown",
		"/tasks/search?page=0",
		"/tasks/search?page_size=nope",
	} {
		t.Run(target, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, target, nil)
			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
		})
	}
}

func TestSearchTasksDefaultsAndCapsPageSize(t *testing.T) {
	server := newTestServer()
	for _, test := range []struct {
		target       string
		wantPageSize int
	}{
		{target: "/tasks/search", wantPageSize: 20},
		{target: "/tasks/search?page_size=200", wantPageSize: 100},
	} {
		req := httptest.NewRequest(http.MethodGet, test.target, nil)
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, req)

		var result store.TaskSearchResult
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatalf("decode %s response: %v", test.target, err)
		}
		if rec.Code != http.StatusOK || result.Page != 1 || result.PageSize != test.wantPageSize {
			t.Fatalf("%s: status=%d page=%d page_size=%d, want 200/1/%d", test.target, rec.Code, result.Page, result.PageSize, test.wantPageSize)
		}
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
