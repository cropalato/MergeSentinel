//
// webservices.go
// Copyright (C) 2023 rmelo <Ricardo Melo <rmelo@ludia.com>>
//
// Distributed under terms of the MIT license.
//

// Package with all handler functions
package webservices

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cropalato/MergeSentinel/internal/conf"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

const (
	actionOpen       = "open"
	actionReopen     = "reopen"
	actionApproved   = "approved"
	actionUnapproved = "unapproved"
)

type Service struct {
	Config     conf.Config `json:"config"`
	HttpClient *http.Client
	DB         *sqlx.DB
}

func (s *Service) updateMergeStatus(project_id int, mr_id int, status string, mr_error string) error {
	mr := []MergeRequestData{}
	query := "SELECT id, target_project_id, iid, description, merge_status, merge_error FROM merge_requests WHERE target_project_id = $1 AND iid = $2"
	err := s.DB.Select(&mr, query, project_id, mr_id)
	if err != nil {
		log.Error().Err(err).Msg("failed to select merge request")
		return err
	}

	var merge_error *string
	if mr_error == "" {
		merge_error = nil
	} else {
		merge_error = &mr_error
	}

	log.Debug().Str("query", query).Any("return", mr).Msg("before update")
	_, err = s.DB.Exec(
		"UPDATE merge_requests SET merge_status = $1, merge_error = $2 WHERE target_project_id = $3 AND iid = $4",
		status, merge_error, project_id, mr_id,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to update merge request")
		return err
	}
	return nil
}

func LoadConfig(cfg_path string) (*Service, error) {
	var s Service
	c, e := conf.NewConfig(cfg_path)
	if e != nil {
		return nil, e
	}
	log.Debug().Str("file", cfg_path).Interface("config", c).Send()

	db, err := sqlx.Connect("postgres", c.PsqlConn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100

	h := &http.Client{
		Timeout:   10 * time.Second,
		Transport: t,
	}
	s = Service{
		Config:     *c,
		HttpClient: h,
		DB:         db,
	}
	return &s, nil
}

func (s *Service) reinforceMrRule(ar conf.ApprovRule, mr_id int) error {
	var approvals GitlabApproval
	log.Debug().Int("project_id", ar.ProjectId).Int("mr", mr_id).Msg("reinforcing MR rule")
	url := fmt.Sprintf("%s/api/v4/projects/%d/merge_requests/%d/approvals", strings.Trim(s.Config.GitlabURL, "/"), ar.ProjectId, mr_id)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Err(err).Send()
		return err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("PRIVATE-TOKEN", s.Config.GitlabToken)
	log.Debug().Str("url", req.URL.String()).Msg("calling gitlab")
	resp, err := s.HttpClient.Do(req)
	if err != nil {
		log.Err(err).Send()
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Err(err).Send()
		return err
	}
	if err := json.Unmarshal(body, &approvals); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal approval response")
		return err
	}
	pending := ar.MinApprov
	for _, by := range approvals.ApprovedBy {
		for _, a := range ar.Approvals {
			if a == by.User.Username {
				pending--
				break
			}
		}
	}
	if pending <= 0 {
		log.Debug().Int("project_id", ar.ProjectId).Int("mr", mr_id).Msg("ok to be merged")
		return s.updateMergeStatus(ar.ProjectId, mr_id, "can_be_merged", "")
	}
	log.Debug().Int("project_id", ar.ProjectId).Int("mr", mr_id).Msg("not ready to be merged")
	msg := fmt.Sprintf("Requires at least %d approvals from %v", ar.MinApprov, ar.Approvals)
	return s.updateMergeStatus(ar.ProjectId, mr_id, "cannot_be_merged", msg)
}

func (s *Service) ReinforceAllMrRule() error {
	var mrList []GitlabMR
	for _, p := range s.Config.Projects {
		log.Debug().Int("project_id", p.ProjectId).Msg("reinforcing MR rule")
		url := fmt.Sprintf("%s/api/v4/projects/%d/merge_requests", strings.Trim(s.Config.GitlabURL, "/"), p.ProjectId)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Err(err).Send()
			continue
		}
		req.Header.Add("Accept", "application/json")
		req.Header.Add("PRIVATE-TOKEN", s.Config.GitlabToken)
		req.URL.Query().Add("state", "opened")
		req.URL.RawQuery = req.URL.Query().Encode()
		log.Debug().Str("url", req.URL.String()).Msg("calling gitlab")
		resp, err := s.HttpClient.Do(req)
		if err != nil {
			log.Err(err).Send()
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Err(err).Send()
			continue
		}
		if err := json.Unmarshal(body, &mrList); err != nil {
			log.Error().Err(err).Msg("failed to unmarshal MR list")
			continue
		}
		for _, mr := range mrList {
			err := s.reinforceMrRule(p, mr.Iid)
			if err != nil {
				log.Err(err).Send()
				continue
			}
		}
	}
	log.Debug().Msg("all MR rules reinforced")
	return nil
}

// State is used to check is the service is running and health.
func (s *Service) State(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", s.Config.CorsOrigin)
	if r.Method == http.MethodOptions {
		return
	}
	w.WriteHeader(200)
	_, err := w.Write([]byte("Service is ready"))
	if err != nil {
		log.Err(err).Send()
	}
}

// PostApproval validate if MR has enough approvals.
// It will return unauthorized http code if rule do not match the required condition.
func (s *Service) PostApproval(w http.ResponseWriter, r *http.Request) {
	var tmp_token string
	w.Header().Set("Access-Control-Allow-Origin", s.Config.CorsOrigin)
	if r.Method == http.MethodOptions {
		return
	}
	request_token, exist := r.Header["X-Gitlab-Token"]
	if !exist {
		if s.Config.WebHookToken != "" {
			http.Error(w, "missing X-Gitlab-Token header", http.StatusUnauthorized)
			return
		}
		log.Warn().Msg("Missing 'X-Gitlab-Token' header.")
	}
	var callback GitlabMREventWebhookCallback
	err := json.NewDecoder(r.Body).Decode(&callback)
	if err != nil {
		log.Err(err).Send()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cb_obj := callback.ObjectKind
	cb_action := callback.ObjectAttributes.Action
	cb_mr_id := callback.ObjectAttributes.Iid
	cb_project := callback.ObjectAttributes.TargetProjectID
	cm_user := callback.User.Username
	log.Debug().Str("user", cm_user).Str("action", cb_action).Str("object", cb_obj).Int("project", cb_project).Int("mr_id", cb_mr_id).Msg("Callback received")
	if cb_action == actionOpen || cb_action == actionReopen || cb_action == actionApproved || cb_action == actionUnapproved {
		for _, p := range s.Config.Projects {
			log.Debug().Int("p.ProjectId", p.ProjectId).Int("cb_project", cb_project).Send()
			if p.ProjectId == cb_project {
				tmp_token = p.WebHookToken
				if tmp_token == "" {
					tmp_token = s.Config.WebHookToken
				}
				if (request_token[0] != "" && tmp_token != "" && request_token[0] != tmp_token) ||
					(request_token[0] == "" && tmp_token != "") {
					err_msg := "mismatching webhook and local tokens."
					err := errors.New(err_msg)
					log.Error().Err(err).Send()
					s.updateMergeStatus(cb_project, cb_mr_id, "cannot_be_merged", err_msg)
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if request_token[0] != "" && tmp_token == "" {
					log.Warn().Msg("Callback with 'X-Gitlab-Token' header, but missing local token config to validate.")
				}
				s.reinforceMrRule(p, cb_mr_id)
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	_, err = w.Write([]byte("{ \"msg\": \"Merge event received\" }\n"))
	if err != nil {
		log.Err(err).Send()
	}
}
