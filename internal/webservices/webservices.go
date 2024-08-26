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

type Service struct {
	Config conf.Config `json:"config"`
	HttpClient *http.Client
}


func (s *Service) updateMergeStatus(project_id int, mr_id int, status string, mr_error string) {
	var merge_error string
	db, err := sqlx.Connect("postgres", s.Config.PsqlConn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
  defer db.Close()

  err = db.Ping()
  if err != nil {
		log.Fatal().Err(err).Send()
  }
	mr := []MergeRequestData{}
	query := fmt.Sprintf("SELECT id, target_project_id, iid, description, merge_status, merge_error FROM merge_requests WHERE target_project_id = %d and iid = %d", project_id, mr_id)
	err = db.Select(&mr, query)
  if err != nil {
		log.Fatal().Err(err).Send()
  }
	if mr_error == "" {
		merge_error = "NULL"
	} else {
		merge_error = fmt.Sprintf("'%s'", mr_error)
	}
	log.Debug().Str("query",query).Any("return",mr).Msg("before update")
	query = fmt.Sprintf("UPDATE %s SET %s = '%s', %s = %s WHERE %s = %d AND %s = %d",
	                    "merge_requests", "merge_status", status, "merge_error", merge_error,
											"target_project_id", project_id, "iid", mr_id)
	_, err = db.Exec(query)
  if err != nil {
		log.Fatal().Err(err).Send()
  }
}


func LoadConfig(cfg_path string) (*Service, error) {
	var s Service
	c, e := conf.NewConfig(cfg_path)
	log.Debug().Str("file", cfg_path).Interface("config", c).Send()
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100

	h := &http.Client{
		Timeout:   10 * time.Second,
		Transport: t,
	}
	s = Service{
		Config: *c,
		HttpClient: h,
	}
	return &s, e
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
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Err(err).Send()
			return err
		}
		if err := json.Unmarshal(body, &approvals); err != nil {
			log.Fatal().Err(err).Send()
		}
		resp.Body.Close()
		pending := ar.MinApprov
		for _, by := range approvals.ApprovedBy {
		  for _, a := range ar.Approvals {
				if a == by.User.Username {
					pending--
					break
				}
			}
		}
		if pending == 0 {
			log.Debug().Int("project_id", ar.ProjectId).Int("mr", mr_id).Msg("ok to be merged")
			s.updateMergeStatus(ar.ProjectId, mr_id, "can_be_merged", "")
		} else {
			log.Debug().Int("project_id", ar.ProjectId).Int("mr", mr_id).Msg("not ready to be merged")
			msg := fmt.Sprintf("Requires at least %d approvals from %v", ar.MinApprov, ar.Approvals)
			s.updateMergeStatus(ar.ProjectId, mr_id, "cannot_be_merged", msg)
		}
	return nil
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
		if err != nil {
			log.Err(err).Send()
			continue
		}
		if err := json.Unmarshal(body, &mrList); err != nil {
			log.Fatal().Err(err).Send()
		}
		resp.Body.Close()
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
	// TODO: Add code to validate if service is ready to reply requests
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
	// w.Header().Set("Access-Control-Allow-Origin", c.CorsOrigin)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		return
	}
	request_token, exist := r.Header["X-Gitlab-Token"]
	if ! exist {
		log.Warn().Msg("Missing 'X-Gitlab-Token' header.")
	}
	var callback GitlabMREventWebhookCallback
	err := json.NewDecoder(r.Body).Decode(&callback)
	if err != nil {
		log.Err(err).Send()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cb_obj    := callback.ObjectKind
	cb_action := callback.ObjectAttributes.Action
	cb_mr_id := callback.ObjectAttributes.Iid
	cb_project := callback.ObjectAttributes.TargetProjectID
	cm_user := callback.User.Username
	log.Debug().Str("user", cm_user).Str("action", cb_action).Str("object", cb_obj).Int("project", cb_project).Int("mr_id", cb_mr_id).Msg("Callback received")
	if cb_action == "open" || cb_action == "reopen" || cb_action == "approved" || cb_action == "unapproved" {
		for _, p := range s.Config.Projects {
			log.Debug().Int("p.ProjectId",p.ProjectId).Int("cb_project",cb_project).Send()
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
					log.Warn().Msg("Callback with 'X-Gitlab-Token' header, but missing local token config to validate." )
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
