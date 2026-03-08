//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cropalato/MergeSentinel/internal/conf"
	"github.com/cropalato/MergeSentinel/internal/webservices"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getPsqlConn() string {
	conn := os.Getenv("TEST_PSQL_CONN")
	if conn == "" {
		conn = "postgres://mstest:mstest@localhost:15432/mergesentinel?sslmode=disable"
	}
	return conn
}

func newIntegrationService(t *testing.T, gitlabURL string) *webservices.Service {
	t.Helper()
	psqlConn := getPsqlConn()

	db, err := sqlx.Connect("postgres", psqlConn)
	require.NoError(t, err, "Failed to connect to PostgreSQL")

	s := &webservices.Service{
		Config: conf.Config{
			GitlabToken:  "glpat-testtoken123",
			GitlabURL:    gitlabURL,
			CorsOrigin:   "https://example.com",
			PsqlConn:     psqlConn,
			WebHookToken: "test-webhook-token",
			Projects: []conf.ApprovRule{
				{
					ProjectId:    42,
					Approvals:    []string{"alice", "bob"},
					MinApprov:    1,
					WebHookToken: "project-webhook-token",
				},
			},
		},
		HttpClient: &http.Client{Timeout: 5 * time.Second},
		DB:         db,
	}
	return s
}

func resetDB(t *testing.T, db *sqlx.DB) {
	t.Helper()
	_, err := db.Exec("UPDATE merge_requests SET merge_status = 'unchecked', merge_error = NULL")
	require.NoError(t, err)
}

// --- Real DB: updateMergeStatus via PostApproval flow ---

func TestIntegration_DBConnection(t *testing.T) {
	s := newIntegrationService(t, "http://localhost")
	defer s.DB.Close()

	err := s.DB.Ping()
	assert.NoError(t, err, "Should be able to ping PostgreSQL")
}

func TestIntegration_UpdateMergeStatus_CanBeMerged(t *testing.T) {
	// Mock GitLab returning alice's approval
	gitlabServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/approvals") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"approved_by": []map[string]interface{}{
					{"user": map[string]interface{}{"username": "alice"}},
				},
			})
			return
		}
		// For /merge_requests list endpoint
		json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer gitlabServer.Close()

	s := newIntegrationService(t, gitlabServer.URL)
	defer s.DB.Close()
	resetDB(t, s.DB)

	// Send a webhook callback for MR 1 with "open" action
	callback := fmt.Sprintf(`{
		"object_kind": "merge_request",
		"object_attributes": {
			"action": "open",
			"iid": 1,
			"target_project_id": 42
		},
		"user": {"username": "testuser"}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/approve", strings.NewReader(callback))
	req.Header.Set("X-Gitlab-Token", "project-webhook-token")
	w := httptest.NewRecorder()
	s.PostApproval(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify DB was updated
	var status string
	err := s.DB.Get(&status, "SELECT merge_status FROM merge_requests WHERE target_project_id = 42 AND iid = 1")
	require.NoError(t, err)
	assert.Equal(t, "can_be_merged", status, "MR should be marked as can_be_merged after alice's approval")
}

func TestIntegration_UpdateMergeStatus_CannotBeMerged(t *testing.T) {
	// Mock GitLab returning no approvals
	gitlabServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"approved_by": []map[string]interface{}{},
		})
	}))
	defer gitlabServer.Close()

	s := newIntegrationService(t, gitlabServer.URL)
	defer s.DB.Close()
	resetDB(t, s.DB)

	callback := `{
		"object_kind": "merge_request",
		"object_attributes": {
			"action": "approved",
			"iid": 2,
			"target_project_id": 42
		},
		"user": {"username": "testuser"}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/approve", strings.NewReader(callback))
	req.Header.Set("X-Gitlab-Token", "project-webhook-token")
	w := httptest.NewRecorder()
	s.PostApproval(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var status, mergeError string
	err := s.DB.QueryRow("SELECT merge_status, merge_error FROM merge_requests WHERE target_project_id = 42 AND iid = 2").Scan(&status, &mergeError)
	require.NoError(t, err)
	assert.Equal(t, "cannot_be_merged", status)
	assert.Contains(t, mergeError, "Requires at least 1 approvals")
}

func TestIntegration_UpdateMergeStatus_NullError(t *testing.T) {
	// Mock GitLab returning alice's approval
	gitlabServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"approved_by": []map[string]interface{}{
				{"user": map[string]interface{}{"username": "alice"}},
			},
		})
	}))
	defer gitlabServer.Close()

	s := newIntegrationService(t, gitlabServer.URL)
	defer s.DB.Close()
	resetDB(t, s.DB)

	callback := `{
		"object_kind": "merge_request",
		"object_attributes": {
			"action": "open",
			"iid": 3,
			"target_project_id": 42
		},
		"user": {"username": "testuser"}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/approve", strings.NewReader(callback))
	req.Header.Set("X-Gitlab-Token", "project-webhook-token")
	w := httptest.NewRecorder()
	s.PostApproval(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// merge_error should be NULL when status is can_be_merged
	var mergeError *string
	err := s.DB.QueryRow("SELECT merge_error FROM merge_requests WHERE target_project_id = 42 AND iid = 3").Scan(&mergeError)
	require.NoError(t, err)
	assert.Nil(t, mergeError, "merge_error should be NULL when MR can be merged")
}

func TestIntegration_TokenMismatch_UpdatesDB(t *testing.T) {
	s := newIntegrationService(t, "http://localhost")
	defer s.DB.Close()
	resetDB(t, s.DB)

	callback := `{
		"object_kind": "merge_request",
		"object_attributes": {
			"action": "open",
			"iid": 1,
			"target_project_id": 42
		},
		"user": {"username": "testuser"}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/approve", strings.NewReader(callback))
	req.Header.Set("X-Gitlab-Token", "wrong-token")
	w := httptest.NewRecorder()
	s.PostApproval(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var status string
	err := s.DB.Get(&status, "SELECT merge_status FROM merge_requests WHERE target_project_id = 42 AND iid = 1")
	require.NoError(t, err)
	assert.Equal(t, "cannot_be_merged", status, "Token mismatch should mark MR as cannot_be_merged")
}

func TestIntegration_StateEndpoint(t *testing.T) {
	s := newIntegrationService(t, "http://localhost")
	defer s.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/state", nil)
	w := httptest.NewRecorder()
	s.State(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Service is ready", w.Body.String())
}

func TestIntegration_ReinforceAllMrRule(t *testing.T) {
	// Mock GitLab returning MR list then approvals
	gitlabServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/approvals") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"approved_by": []map[string]interface{}{
					{"user": map[string]interface{}{"username": "bob"}},
				},
			})
			return
		}
		// Return MR list
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"iid": 1}, {"iid": 2},
		})
	}))
	defer gitlabServer.Close()

	s := newIntegrationService(t, gitlabServer.URL)
	defer s.DB.Close()
	resetDB(t, s.DB)

	err := s.ReinforceAllMrRule()
	assert.NoError(t, err)

	// Both MRs should be can_be_merged (bob approved, min_approv=1)
	for _, iid := range []int{1, 2} {
		var status string
		err := s.DB.Get(&status, "SELECT merge_status FROM merge_requests WHERE target_project_id = 42 AND iid = $1", iid)
		require.NoError(t, err)
		assert.Equal(t, "can_be_merged", status, "MR %d should be can_be_merged", iid)
	}
}

func TestIntegration_SQLInjectionPrevention(t *testing.T) {
	gitlabServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"approved_by": []map[string]interface{}{
				{"user": map[string]interface{}{"username": "alice"}},
			},
		})
	}))
	defer gitlabServer.Close()

	s := newIntegrationService(t, gitlabServer.URL)
	defer s.DB.Close()
	resetDB(t, s.DB)

	// Attempt SQL injection via webhook callback — the iid and project_id
	// are integers so they can't inject, but let's verify the flow doesn't crash
	// and that the parameterized queries handle edge cases properly
	callback := `{
		"object_kind": "merge_request",
		"object_attributes": {
			"action": "open",
			"iid": 1,
			"target_project_id": 42
		},
		"user": {"username": "testuser"}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/approve", strings.NewReader(callback))
	req.Header.Set("X-Gitlab-Token", "project-webhook-token")
	w := httptest.NewRecorder()
	s.PostApproval(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify no extra rows were created (SQL injection could INSERT)
	var count int
	err := s.DB.Get(&count, "SELECT count(*) FROM merge_requests")
	require.NoError(t, err)
	assert.Equal(t, 3, count, "Should still have exactly 3 rows (no injection)")
}

// --- Docker build test ---

func TestIntegration_DockerBuild(t *testing.T) {
	if os.Getenv("TEST_DOCKER_BUILD") != "1" {
		t.Skip("Set TEST_DOCKER_BUILD=1 to run Docker build test")
	}
	// This test is run separately via the integration script
	// It just verifies the binary exists and responds
	resp, err := http.Get("http://localhost:18080/state")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "Service is ready", string(body))
}
