package webservices

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/cropalato/MergeSentinel/internal/conf"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (*Service, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	s := &Service{
		Config: conf.Config{
			GitlabToken:  "glpat-test",
			GitlabURL:    "https://gitlab.example.com",
			CorsOrigin:   "https://example.com",
			PsqlConn:     "postgres://test:test@localhost/test",
			WebHookToken: "global-token",
			Projects: []conf.ApprovRule{
				{
					ProjectId:    42,
					Approvals:    []string{"alice", "bob"},
					MinApprov:    1,
					WebHookToken: "project-token",
				},
			},
		},
		HttpClient: &http.Client{},
		DB:         sqlxDB,
	}
	return s, mock
}

// --- updateMergeStatus tests ---

func TestUpdateMergeStatus_ParameterizedSelect(t *testing.T) {
	s, mock := newTestService(t)
	defer s.DB.Close()

	rows := sqlmock.NewRows([]string{"id", "target_project_id", "iid", "description", "merge_status", "merge_error"})
	mock.ExpectQuery(`SELECT .+ FROM merge_requests WHERE target_project_id = \$1 AND iid = \$2`).
		WithArgs(42, 10).
		WillReturnRows(rows)
	mock.ExpectExec(`UPDATE merge_requests SET merge_status = \$1, merge_error = \$2 WHERE target_project_id = \$3 AND iid = \$4`).
		WithArgs("can_be_merged", nil, 42, 10).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.updateMergeStatus(42, 10, "can_be_merged", "")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMergeStatus_WithError(t *testing.T) {
	s, mock := newTestService(t)
	defer s.DB.Close()

	rows := sqlmock.NewRows([]string{"id", "target_project_id", "iid", "description", "merge_status", "merge_error"})
	mock.ExpectQuery(`SELECT .+ FROM merge_requests WHERE`).
		WithArgs(42, 10).
		WillReturnRows(rows)

	errorMsg := "Requires at least 2 approvals from [alice bob]"
	mock.ExpectExec(`UPDATE merge_requests SET`).
		WithArgs("cannot_be_merged", &errorMsg, 42, 10).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.updateMergeStatus(42, 10, "cannot_be_merged", errorMsg)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMergeStatus_SelectError(t *testing.T) {
	s, mock := newTestService(t)
	defer s.DB.Close()

	mock.ExpectQuery(`SELECT .+ FROM merge_requests WHERE`).
		WithArgs(42, 10).
		WillReturnError(fmt.Errorf("connection refused"))

	err := s.updateMergeStatus(42, 10, "can_be_merged", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestUpdateMergeStatus_ExecError(t *testing.T) {
	s, mock := newTestService(t)
	defer s.DB.Close()

	rows := sqlmock.NewRows([]string{"id", "target_project_id", "iid", "description", "merge_status", "merge_error"})
	mock.ExpectQuery(`SELECT .+ FROM merge_requests WHERE`).
		WithArgs(42, 10).
		WillReturnRows(rows)
	mock.ExpectExec(`UPDATE merge_requests SET`).
		WithArgs("can_be_merged", nil, 42, 10).
		WillReturnError(fmt.Errorf("update failed"))

	err := s.updateMergeStatus(42, 10, "can_be_merged", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update failed")
}

// --- State handler tests ---

func TestState_CORSHeader(t *testing.T) {
	s, _ := newTestService(t)
	defer s.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/state", nil)
	w := httptest.NewRecorder()
	s.State(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Service is ready", w.Body.String())
}

func TestState_Options(t *testing.T) {
	s, _ := newTestService(t)
	defer s.DB.Close()

	req := httptest.NewRequest(http.MethodOptions, "/state", nil)
	w := httptest.NewRecorder()
	s.State(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
}

// --- PostApproval handler tests ---

func TestPostApproval_CORSHeader(t *testing.T) {
	s, _ := newTestService(t)
	defer s.DB.Close()

	body := `{"object_kind":"merge_request","object_attributes":{"action":"close","iid":1,"target_project_id":42},"user":{"username":"test"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approve", strings.NewReader(body))
	req.Header.Set("X-Gitlab-Token", "project-token")
	w := httptest.NewRecorder()
	s.PostApproval(w, req)

	assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"),
		"CORS should use configured origin, not wildcard")
}

func TestPostApproval_MissingTokenReturns401(t *testing.T) {
	s, _ := newTestService(t)
	defer s.DB.Close()

	body := `{"object_kind":"merge_request","object_attributes":{"action":"open","iid":1,"target_project_id":42},"user":{"username":"test"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approve", strings.NewReader(body))
	// No X-Gitlab-Token header
	w := httptest.NewRecorder()
	s.PostApproval(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"Should return 401 when X-Gitlab-Token is missing and a global token is configured")
}

func TestPostApproval_MissingTokenAllowedWhenNoGlobalToken(t *testing.T) {
	s, _ := newTestService(t)
	defer s.DB.Close()
	s.Config.WebHookToken = ""

	body := `{"object_kind":"merge_request","object_attributes":{"action":"close","iid":1,"target_project_id":99},"user":{"username":"test"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approve", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.PostApproval(w, req)

	assert.Equal(t, http.StatusOK, w.Code,
		"Should allow request when no global webhook token is configured")
}

func TestPostApproval_InvalidJSON(t *testing.T) {
	s, _ := newTestService(t)
	defer s.DB.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/approve", strings.NewReader("not json"))
	req.Header.Set("X-Gitlab-Token", "project-token")
	w := httptest.NewRecorder()
	s.PostApproval(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostApproval_TokenMismatch(t *testing.T) {
	s, mock := newTestService(t)
	defer s.DB.Close()

	// Expect updateMergeStatus to be called with "cannot_be_merged"
	rows := sqlmock.NewRows([]string{"id", "target_project_id", "iid", "description", "merge_status", "merge_error"})
	mock.ExpectQuery(`SELECT .+ FROM merge_requests WHERE`).
		WithArgs(42, 1).
		WillReturnRows(rows)
	mock.ExpectExec(`UPDATE merge_requests SET`).
		WithArgs("cannot_be_merged", sqlmock.AnyArg(), 42, 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	body := `{"object_kind":"merge_request","object_attributes":{"action":"open","iid":1,"target_project_id":42},"user":{"username":"test"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approve", strings.NewReader(body))
	req.Header.Set("X-Gitlab-Token", "wrong-token")
	w := httptest.NewRecorder()
	s.PostApproval(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostApproval_IgnoresNonMatchingActions(t *testing.T) {
	s, _ := newTestService(t)
	defer s.DB.Close()

	body := `{"object_kind":"merge_request","object_attributes":{"action":"close","iid":1,"target_project_id":42},"user":{"username":"test"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approve", strings.NewReader(body))
	req.Header.Set("X-Gitlab-Token", "project-token")
	w := httptest.NewRecorder()
	s.PostApproval(w, req)

	assert.Equal(t, http.StatusOK, w.Code,
		"Non-matching actions (close) should return 200 without processing")
}

// --- reinforceMrRule tests ---

func TestReinforceMrRule_CanBeMerged(t *testing.T) {
	s, mock := newTestService(t)
	defer s.DB.Close()

	// Mock GitLab API returning one approval from "alice"
	gitlabServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "glpat-test", r.Header.Get("PRIVATE-TOKEN"))
		resp := GitlabApproval{
			ApprovedBy: []struct {
				User struct {
					ID        int    `json:"id"`
					Username  string `json:"username"`
					Name      string `json:"name"`
					State     string `json:"state"`
					Locked    bool   `json:"locked"`
					AvatarURL string `json:"avatar_url"`
					WebURL    string `json:"web_url"`
				} `json:"user"`
			}{
				{User: struct {
					ID        int    `json:"id"`
					Username  string `json:"username"`
					Name      string `json:"name"`
					State     string `json:"state"`
					Locked    bool   `json:"locked"`
					AvatarURL string `json:"avatar_url"`
					WebURL    string `json:"web_url"`
				}{Username: "alice"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer gitlabServer.Close()
	s.Config.GitlabURL = gitlabServer.URL

	// Expect "can_be_merged" since alice approved and min_approv=1
	rows := sqlmock.NewRows([]string{"id", "target_project_id", "iid", "description", "merge_status", "merge_error"})
	mock.ExpectQuery(`SELECT .+ FROM merge_requests WHERE`).
		WithArgs(42, 5).
		WillReturnRows(rows)
	mock.ExpectExec(`UPDATE merge_requests SET`).
		WithArgs("can_be_merged", nil, 42, 5).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.reinforceMrRule(s.Config.Projects[0], 5)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReinforceMrRule_CannotBeMerged(t *testing.T) {
	s, mock := newTestService(t)
	defer s.DB.Close()

	// Update min_approv to require 2 approvals
	s.Config.Projects[0].MinApprov = 2

	// Mock GitLab API returning one approval from "alice"
	gitlabServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := GitlabApproval{
			ApprovedBy: []struct {
				User struct {
					ID        int    `json:"id"`
					Username  string `json:"username"`
					Name      string `json:"name"`
					State     string `json:"state"`
					Locked    bool   `json:"locked"`
					AvatarURL string `json:"avatar_url"`
					WebURL    string `json:"web_url"`
				} `json:"user"`
			}{
				{User: struct {
					ID        int    `json:"id"`
					Username  string `json:"username"`
					Name      string `json:"name"`
					State     string `json:"state"`
					Locked    bool   `json:"locked"`
					AvatarURL string `json:"avatar_url"`
					WebURL    string `json:"web_url"`
				}{Username: "alice"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer gitlabServer.Close()
	s.Config.GitlabURL = gitlabServer.URL

	// Expect "cannot_be_merged" since only 1 of 2 required approvals
	errorMsg := "Requires at least 2 approvals from [alice bob]"
	rows := sqlmock.NewRows([]string{"id", "target_project_id", "iid", "description", "merge_status", "merge_error"})
	mock.ExpectQuery(`SELECT .+ FROM merge_requests WHERE`).
		WithArgs(42, 5).
		WillReturnRows(rows)
	mock.ExpectExec(`UPDATE merge_requests SET`).
		WithArgs("cannot_be_merged", &errorMsg, 42, 5).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.reinforceMrRule(s.Config.Projects[0], 5)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReinforceMrRule_GitlabAPIError(t *testing.T) {
	s, _ := newTestService(t)
	defer s.DB.Close()

	// Point to a server that will close immediately
	s.Config.GitlabURL = "http://127.0.0.1:1"

	err := s.reinforceMrRule(s.Config.Projects[0], 5)
	assert.Error(t, err, "Should return error when GitLab API is unreachable")
}

func TestReinforceMrRule_InvalidJSON(t *testing.T) {
	s, _ := newTestService(t)
	defer s.DB.Close()

	gitlabServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer gitlabServer.Close()
	s.Config.GitlabURL = gitlabServer.URL

	err := s.reinforceMrRule(s.Config.Projects[0], 5)
	assert.Error(t, err, "Should return error on invalid JSON instead of log.Fatal")
}

// --- ReinforceAllMrRule tests ---

func TestReinforceAllMrRule_GitlabAPIError(t *testing.T) {
	s, _ := newTestService(t)
	defer s.DB.Close()

	s.Config.GitlabURL = "http://127.0.0.1:1"

	// Should not crash, just log and continue
	err := s.ReinforceAllMrRule()
	assert.NoError(t, err, "Should gracefully handle API errors without crashing")
}

func TestReinforceAllMrRule_InvalidJSON(t *testing.T) {
	s, _ := newTestService(t)
	defer s.DB.Close()

	gitlabServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer gitlabServer.Close()
	s.Config.GitlabURL = gitlabServer.URL

	// Should not crash (was log.Fatal before)
	err := s.ReinforceAllMrRule()
	assert.NoError(t, err, "Should gracefully handle invalid JSON without crashing")
}

// --- Action constants tests ---

func TestActionConstants(t *testing.T) {
	assert.Equal(t, "open", actionOpen)
	assert.Equal(t, "reopen", actionReopen)
	assert.Equal(t, "approved", actionApproved)
	assert.Equal(t, "unapproved", actionUnapproved)
}
