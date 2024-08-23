//
// type.go
// Copyright (C) 2024 rmelo <Ricardo Melo <rmelo@ludia.com>>
//
// Distributed under terms of the MIT license.
//

package webservices

import (
	"time"
	"database/sql"
)

type GitlabMR struct {
	// Based on gitlab version 17.2
	ID             int         `json:"id"`
	Iid            int         `json:"iid"`
	ProjectID      int         `json:"project_id"`
	Title          string      `json:"title"`
	Description    string      `json:"description"`
	State          string      `json:"state"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	MergedBy       interface{} `json:"merged_by"`
	MergeUser      interface{} `json:"merge_user"`
	MergedAt       interface{} `json:"merged_at"`
	ClosedBy       interface{} `json:"closed_by"`
	ClosedAt       interface{} `json:"closed_at"`
	TargetBranch   string      `json:"target_branch"`
	SourceBranch   string      `json:"source_branch"`
	UserNotesCount int         `json:"user_notes_count"`
	Upvotes        int         `json:"upvotes"`
	Downvotes      int         `json:"downvotes"`
	Author         struct {
		ID        int    `json:"id"`
		Username  string `json:"username"`
		Name      string `json:"name"`
		State     string `json:"state"`
		Locked    bool   `json:"locked"`
		AvatarURL string `json:"avatar_url"`
		WebURL    string `json:"web_url"`
	} `json:"author"`
	Assignees                 []interface{} `json:"assignees"`
	Assignee                  interface{}   `json:"assignee"`
	Reviewers                 []interface{} `json:"reviewers"`
	SourceProjectID           int           `json:"source_project_id"`
	TargetProjectID           int           `json:"target_project_id"`
	Labels                    []interface{} `json:"labels"`
	Draft                     bool          `json:"draft"`
	Imported                  bool          `json:"imported"`
	ImportedFrom              string        `json:"imported_from"`
	WorkInProgress            bool          `json:"work_in_progress"`
	Milestone                 interface{}   `json:"milestone"`
	MergeWhenPipelineSucceeds bool          `json:"merge_when_pipeline_succeeds"`
	MergeStatus               string        `json:"merge_status"`
	DetailedMergeStatus       string        `json:"detailed_merge_status"`
	Sha                       string        `json:"sha"`
	MergeCommitSha            interface{}   `json:"merge_commit_sha"`
	SquashCommitSha           interface{}   `json:"squash_commit_sha"`
	DiscussionLocked          interface{}   `json:"discussion_locked"`
	ShouldRemoveSourceBranch  interface{}   `json:"should_remove_source_branch"`
	ForceRemoveSourceBranch   bool          `json:"force_remove_source_branch"`
	PreparedAt                time.Time     `json:"prepared_at"`
	Reference                 string        `json:"reference"`
	References                struct {
		Short    string `json:"short"`
		Relative string `json:"relative"`
		Full     string `json:"full"`
	} `json:"references"`
	WebURL    string `json:"web_url"`
	TimeStats struct {
		TimeEstimate        int         `json:"time_estimate"`
		TotalTimeSpent      int         `json:"total_time_spent"`
		HumanTimeEstimate   interface{} `json:"human_time_estimate"`
		HumanTotalTimeSpent interface{} `json:"human_total_time_spent"`
	} `json:"time_stats"`
	Squash               bool `json:"squash"`
	SquashOnMerge        bool `json:"squash_on_merge"`
	TaskCompletionStatus struct {
		Count          int `json:"count"`
		CompletedCount int `json:"completed_count"`
	} `json:"task_completion_status"`
	HasConflicts                bool `json:"has_conflicts"`
	BlockingDiscussionsResolved bool `json:"blocking_discussions_resolved"`
}
type GitlabApproval struct {
	// Based on gitlab version 17.2
	UserHasApproved bool `json:"user_has_approved"`
	UserCanApprove  bool `json:"user_can_approve"`
	Approved        bool `json:"approved"`
	ApprovedBy      []struct {
		User struct {
			ID        int    `json:"id"`
			Username  string `json:"username"`
			Name      string `json:"name"`
			State     string `json:"state"`
			Locked    bool   `json:"locked"`
			AvatarURL string `json:"avatar_url"`
			WebURL    string `json:"web_url"`
		} `json:"user"`
	} `json:"approved_by"`
}

type GitlabMREventWebhookCallback struct {
	// Based on gitlab version 17.2
	Changes struct {
		UpdatedAt struct {
			Current  string `json:"current"`
			Previous string `json:"previous"`
		} `json:"updated_at"`
	} `json:"changes"`
	EventType        string        `json:"event_type"`
	Labels           []interface{} `json:"labels"`
	ObjectAttributes struct {
		Action                      string        `json:"action"`
		AssigneeID                  interface{}   `json:"assignee_id"`
		AssigneeIds                 []interface{} `json:"assignee_ids"`
		AuthorID                    int           `json:"author_id"`
		BlockingDiscussionsResolved bool          `json:"blocking_discussions_resolved"`
		CreatedAt                   string        `json:"created_at"`
		Description                 string        `json:"description"`
		DetailedMergeStatus         string        `json:"detailed_merge_status"`
		Draft                       bool          `json:"draft"`
		FirstContribution           bool          `json:"first_contribution"`
		HeadPipelineID              interface{}   `json:"head_pipeline_id"`
		HumanTimeChange             interface{}   `json:"human_time_change"`
		HumanTimeEstimate           interface{}   `json:"human_time_estimate"`
		HumanTotalTimeSpent         interface{}   `json:"human_total_time_spent"`
		ID                          int           `json:"id"`
		Iid                         int           `json:"iid"`
		Labels                      []interface{} `json:"labels"`
		LastCommit                  struct {
			Author struct {
				Email string `json:"email"`
				Name  string `json:"name"`
			} `json:"author"`
			ID        string `json:"id"`
			Message   string `json:"message"`
			Timestamp string `json:"timestamp"`
			Title     string `json:"title"`
			URL       string `json:"url"`
		} `json:"last_commit"`
		LastEditedAt   interface{} `json:"last_edited_at"`
		LastEditedByID interface{} `json:"last_edited_by_id"`
		MergeCommitSha interface{} `json:"merge_commit_sha"`
		MergeError     string      `json:"merge_error"`
		MergeParams    struct {
			ForceRemoveSourceBranch string `json:"force_remove_source_branch"`
		} `json:"merge_params"`
		MergeStatus               string        `json:"merge_status"`
		MergeUserID               interface{}   `json:"merge_user_id"`
		MergeWhenPipelineSucceeds bool          `json:"merge_when_pipeline_succeeds"`
		MilestoneID               interface{}   `json:"milestone_id"`
		PreparedAt                string        `json:"prepared_at"`
		ReviewerIds               []interface{} `json:"reviewer_ids"`
		Source                    struct {
			AvatarURL         interface{} `json:"avatar_url"`
			CiConfigPath      interface{} `json:"ci_config_path"`
			DefaultBranch     string      `json:"default_branch"`
			Description       string      `json:"description"`
			GitHTTPURL        string      `json:"git_http_url"`
			GitSSHURL         string      `json:"git_ssh_url"`
			Homepage          string      `json:"homepage"`
			HTTPURL           string      `json:"http_url"`
			ID                int         `json:"id"`
			Name              string      `json:"name"`
			Namespace         string      `json:"namespace"`
			PathWithNamespace string      `json:"path_with_namespace"`
			SSHURL            string      `json:"ssh_url"`
			URL               string      `json:"url"`
			VisibilityLevel   int         `json:"visibility_level"`
			WebURL            string      `json:"web_url"`
		} `json:"source"`
		SourceBranch    string `json:"source_branch"`
		SourceProjectID int    `json:"source_project_id"`
		State           string `json:"state"`
		StateID         int    `json:"state_id"`
		Target          struct {
			AvatarURL         interface{} `json:"avatar_url"`
			CiConfigPath      interface{} `json:"ci_config_path"`
			DefaultBranch     string      `json:"default_branch"`
			Description       string      `json:"description"`
			GitHTTPURL        string      `json:"git_http_url"`
			GitSSHURL         string      `json:"git_ssh_url"`
			Homepage          string      `json:"homepage"`
			HTTPURL           string      `json:"http_url"`
			ID                int         `json:"id"`
			Name              string      `json:"name"`
			Namespace         string      `json:"namespace"`
			PathWithNamespace string      `json:"path_with_namespace"`
			SSHURL            string      `json:"ssh_url"`
			URL               string      `json:"url"`
			VisibilityLevel   int         `json:"visibility_level"`
			WebURL            string      `json:"web_url"`
		} `json:"target"`
		TargetBranch    string `json:"target_branch"`
		TargetProjectID int    `json:"target_project_id"`
		TimeChange      int    `json:"time_change"`
		TimeEstimate    int    `json:"time_estimate"`
		Title           string `json:"title"`
		TotalTimeSpent  int    `json:"total_time_spent"`
		UpdatedAt       string `json:"updated_at"`
		UpdatedByID     int    `json:"updated_by_id"`
		URL             string `json:"url"`
		WorkInProgress  bool   `json:"work_in_progress"`
	} `json:"object_attributes"`
	ObjectKind string `json:"object_kind"`
	Project    struct {
		AvatarURL         interface{} `json:"avatar_url"`
		CiConfigPath      interface{} `json:"ci_config_path"`
		DefaultBranch     string      `json:"default_branch"`
		Description       string      `json:"description"`
		GitHTTPURL        string      `json:"git_http_url"`
		GitSSHURL         string      `json:"git_ssh_url"`
		Homepage          string      `json:"homepage"`
		HTTPURL           string      `json:"http_url"`
		ID                int         `json:"id"`
		Name              string      `json:"name"`
		Namespace         string      `json:"namespace"`
		PathWithNamespace string      `json:"path_with_namespace"`
		SSHURL            string      `json:"ssh_url"`
		URL               string      `json:"url"`
		VisibilityLevel   int         `json:"visibility_level"`
		WebURL            string      `json:"web_url"`
	} `json:"project"`
	Repository struct {
		Description string `json:"description"`
		Homepage    string `json:"homepage"`
		Name        string `json:"name"`
		URL         string `json:"url"`
	} `json:"repository"`
	User struct {
		AvatarURL string `json:"avatar_url"`
		Email     string `json:"email"`
		ID        int    `json:"id"`
		Name      string `json:"name"`
		Username  string `json:"username"`
	} `json:"user"`
}

type MergeRequestData struct {
	Id int `db:"id"`
	TargetProjectId int `db:"target_project_id"`
	Iid int `db:"iid"`
	Description sql.NullString `db:"description"`
	MergeStatus string `db:"merge_status"` 
	MergeError sql.NullString `db:"merge_error"`
}
