package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/AxeForging/reviewforge/domain"
	"github.com/AxeForging/reviewforge/helpers"
	"github.com/rs/zerolog/log"
)

// GitHubService handles all GitHub REST API interactions
type GitHubService struct {
	Token   string
	Owner   string
	Repo    string
	BaseURL string // override for testing; defaults to https://api.github.com
}

// NewGitHubService creates a GitHubService from a token and owner/repo string
func NewGitHubService(token, repo string) (*GitHubService, error) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("%w: got %q", helpers.ErrNoRepo, repo)
	}
	return &GitHubService{
		Token: token,
		Owner: parts[0],
		Repo:  parts[1],
	}, nil
}

func (s *GitHubService) baseURL() string {
	if s.BaseURL != "" {
		return s.BaseURL
	}
	return "https://api.github.com"
}

func (s *GitHubService) doRequest(method, path string, body io.Reader, accept string) ([]byte, int, error) {
	url := s.baseURL() + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, 0, helpers.WrapError(err, "github", "failed to create request")
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if accept != "" {
		req.Header.Set("Accept", accept)
	} else {
		req.Header.Set("Accept", "application/vnd.github+json")
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, helpers.WrapError(err, "github", "request failed")
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, helpers.WrapError(err, "github", "failed to read response")
	}

	return data, resp.StatusCode, nil
}

// GetPRDetails fetches pull request metadata
func (s *GitHubService) GetPRDetails(prNumber int) (*domain.PRDetails, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", s.Owner, s.Repo, prNumber)
	data, status, err := s.doRequest("GET", path, nil, "")
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("%w: GET %s returned %d: %s", helpers.ErrGitHubAPI, path, status, string(data))
	}

	var pr struct {
		Title string `json:"title"`
		Body  string `json:"body"`
		Base  struct {
			SHA string `json:"sha"`
		} `json:"base"`
		Head struct {
			SHA string `json:"sha"`
		} `json:"head"`
	}
	if err := json.Unmarshal(data, &pr); err != nil {
		return nil, helpers.WrapError(err, "github", "failed to parse PR details")
	}

	return &domain.PRDetails{
		Owner:       s.Owner,
		Repo:        s.Repo,
		Number:      prNumber,
		Title:       pr.Title,
		Description: pr.Body,
		BaseSHA:     pr.Base.SHA,
		HeadSHA:     pr.Head.SHA,
	}, nil
}

// GetPRDiff fetches the unified diff for a PR or compare range
func (s *GitHubService) GetPRDiff(prNumber int, baseCommit string) (string, error) {
	var path string
	if baseCommit != "" {
		// Incremental: compare from last reviewed commit to current HEAD
		path = fmt.Sprintf("/repos/%s/%s/compare/%s...HEAD", s.Owner, s.Repo, baseCommit)
	} else {
		path = fmt.Sprintf("/repos/%s/%s/pulls/%d", s.Owner, s.Repo, prNumber)
	}

	data, status, err := s.doRequest("GET", path, nil, "application/vnd.github.v3.diff")
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("%w: GET diff returned %d: %s", helpers.ErrGitHubAPI, status, string(data))
	}

	return string(data), nil
}

// GetFileContent fetches file content at a given ref (SHA or branch)
func (s *GitHubService) GetFileContent(path, ref string) (string, error) {
	apiPath := fmt.Sprintf("/repos/%s/%s/contents/%s", s.Owner, s.Repo, path)
	if ref != "" {
		apiPath += "?ref=" + ref
	}

	data, status, err := s.doRequest("GET", apiPath, nil, "")
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		log.Debug().Str("path", path).Int("status", status).Msg("Failed to fetch file content")
		return "", nil
	}

	var file struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return "", helpers.WrapError(err, "github", "failed to parse file content response")
	}

	if file.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(file.Content, "\n", ""))
		if err != nil {
			return "", helpers.WrapError(err, "github", "failed to decode base64 content")
		}
		return string(decoded), nil
	}

	return file.Content, nil
}

// GetLastReviewedCommit finds the commit SHA of the last bot review on the PR
func (s *GitHubService) GetLastReviewedCommit(prNumber int) (string, error) {
	// Get all reviews
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", s.Owner, s.Repo, prNumber)
	data, status, err := s.doRequest("GET", path, nil, "")
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("%w: GET reviews returned %d", helpers.ErrGitHubAPI, status)
	}

	var reviews []struct {
		User struct {
			Login string `json:"login"`
		} `json:"user"`
		CommitID    string `json:"commit_id"`
		SubmittedAt string `json:"submitted_at"`
	}
	if err := json.Unmarshal(data, &reviews); err != nil {
		return "", helpers.WrapError(err, "github", "failed to parse reviews")
	}

	// Find last bot review (reverse order)
	for i := len(reviews) - 1; i >= 0; i-- {
		if reviews[i].User.Login == "github-actions[bot]" {
			return reviews[i].CommitID, nil
		}
	}

	return "", nil
}

// SubmitReview posts a review with line comments to the PR
func (s *GitHubService) SubmitReview(prNumber int, output *domain.AIReviewOutput, event string) error {
	comments := make([]map[string]interface{}, 0, len(output.Comments))
	for _, c := range output.Comments {
		severity := ""
		if c.Severity != "" {
			severity = fmt.Sprintf("**[%s]** ", strings.ToUpper(c.Severity))
		}
		comments = append(comments, map[string]interface{}{
			"path": c.Path,
			"line": c.Line,
			"side": "RIGHT",
			"body": severity + c.Comment,
		})
	}

	body := map[string]interface{}{
		"body":     output.Summary,
		"event":    strings.ToUpper(event),
		"comments": comments,
	}

	return s.postReview(prNumber, body, comments)
}

func (s *GitHubService) postReview(prNumber int, body map[string]interface{}, comments []map[string]interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return helpers.WrapError(err, "github", "failed to marshal review")
	}

	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", s.Owner, s.Repo, prNumber)
	respData, status, err := s.doRequest("POST", path, strings.NewReader(string(data)), "")
	if err != nil {
		return err
	}

	if status == http.StatusOK || status == http.StatusCreated {
		log.Info().Int("comments", len(comments)).Msg("Review submitted successfully")
		return nil
	}

	// 422 often means line comments are invalid or event not allowed — retry with fallbacks
	if status == http.StatusUnprocessableEntity {
		// First fallback: downgrade event to COMMENT (can't approve/request_changes on own PR)
		if body["event"] != "COMMENT" {
			log.Warn().Str("event", body["event"].(string)).Msg("Review submission failed, retrying with COMMENT event")
			body["event"] = "COMMENT"

			retryData, _ := json.Marshal(body)
			respData, status, err = s.doRequest("POST", path, strings.NewReader(string(retryData)), "")
			if err != nil {
				return err
			}
			if status == http.StatusOK || status == http.StatusCreated {
				log.Info().Int("comments", len(comments)).Msg("Review submitted with COMMENT event")
				return nil
			}
		}

		// Second fallback: remove line comments (invalid line numbers, etc.)
		if len(comments) > 0 {
			log.Warn().Int("status", status).Msg("Review submission still failing, retrying without line comments")

			body["comments"] = []map[string]interface{}{}

			retryData, _ := json.Marshal(body)
			respData, status, err = s.doRequest("POST", path, strings.NewReader(string(retryData)), "")
			if err != nil {
				return err
			}
			if status == http.StatusOK || status == http.StatusCreated {
				log.Info().Msg("Review submitted without line comments")
				return nil
			}
		}
	}

	return fmt.Errorf("%w: POST review returned %d: %s", helpers.ErrGitHubAPI, status, string(respData))
}
