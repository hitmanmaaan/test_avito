package apperrors

import "fmt"

type ServiceError struct {
	Code    string
	Message string
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

var (
	ErrTeamExists  = &ServiceError{Code: "TEAM_EXISTS", Message: "team_name already exists"}
	ErrPRExists    = &ServiceError{Code: "PR_EXISTS", Message: "PR id already exists"}
	ErrPRMerged    = &ServiceError{Code: "PR_MERGED", Message: "cannot modify reviewers on merged PR"}
	ErrNotAssigned = &ServiceError{Code: "NOT_ASSIGNED", Message: "reviewer is not assigned to this PR"}
	ErrNoCandidate = &ServiceError{Code: "NO_CANDIDATE", Message: "no active replacement candidate in team"}
	ErrNotFound    = &ServiceError{Code: "NOT_FOUND", Message: "resource not found"}
)
