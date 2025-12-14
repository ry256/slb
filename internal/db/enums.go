// Package db provides database types and operations for SLB.
package db

import "errors"

// RiskTier represents the risk classification of a command.
type RiskTier string

const (
	// RiskTierCritical requires 2+ approvals (data destruction, production deploys).
	RiskTierCritical RiskTier = "critical"
	// RiskTierDangerous requires 1 approval (force pushes, schema changes).
	RiskTierDangerous RiskTier = "dangerous"
	// RiskTierCaution is auto-approved after 30s with notification.
	RiskTierCaution RiskTier = "caution"
)

// Valid returns true if the tier is a valid risk tier.
func (t RiskTier) Valid() bool {
	switch t {
	case RiskTierCritical, RiskTierDangerous, RiskTierCaution:
		return true
	default:
		return false
	}
}

// MinApprovals returns the minimum approvals required for this tier.
func (t RiskTier) MinApprovals() int {
	switch t {
	case RiskTierCritical:
		return 2
	case RiskTierDangerous:
		return 1
	case RiskTierCaution:
		return 0
	default:
		return 2 // Default to most restrictive
	}
}

// RequestStatus represents the current state of a request.
type RequestStatus string

const (
	// StatusPending means the request is waiting for approval.
	StatusPending RequestStatus = "pending"
	// StatusApproved means the request has been approved but not executed.
	StatusApproved RequestStatus = "approved"
	// StatusRejected means the request has been rejected.
	StatusRejected RequestStatus = "rejected"
	// StatusExecuting means the command is currently being executed.
	StatusExecuting RequestStatus = "executing"
	// StatusExecuted means the command was executed successfully.
	StatusExecuted RequestStatus = "executed"
	// StatusExecutionFailed means the command failed during execution.
	StatusExecutionFailed RequestStatus = "execution_failed"
	// StatusCancelled means the request was cancelled by the requestor.
	StatusCancelled RequestStatus = "cancelled"
	// StatusTimeout means the request timed out waiting for approval.
	StatusTimeout RequestStatus = "timeout"
	// StatusTimedOut means the command timed out during execution.
	StatusTimedOut RequestStatus = "timed_out"
	// StatusEscalated means the request was escalated (e.g., caution -> dangerous).
	StatusEscalated RequestStatus = "escalated"
)

// Valid returns true if the status is a valid request status.
func (s RequestStatus) Valid() bool {
	switch s {
	case StatusPending, StatusApproved, StatusRejected, StatusExecuting, StatusExecuted,
		StatusExecutionFailed, StatusCancelled, StatusTimeout, StatusTimedOut,
		StatusEscalated:
		return true
	default:
		return false
	}
}

// IsTerminal returns true if the status is a terminal state.
func (s RequestStatus) IsTerminal() bool {
	switch s {
	case StatusExecuted, StatusExecutionFailed, StatusCancelled, StatusRejected,
		StatusTimedOut:
		return true
	default:
		return false
	}
}

// IsPending returns true if the request is waiting for action.
func (s RequestStatus) IsPending() bool {
	return s == StatusPending || s == StatusApproved
}

// Decision represents an approval or rejection decision.
type Decision string

const (
	// DecisionApprove means the reviewer approved the request.
	DecisionApprove Decision = "approve"
	// DecisionReject means the reviewer rejected the request.
	DecisionReject Decision = "reject"
)

// ErrReviewNotFound indicates a missing review.
var ErrReviewNotFound = errors.New("review not found")

// Valid returns true if the decision is valid.
func (d Decision) Valid() bool {
	return d == DecisionApprove || d == DecisionReject
}

// AttachmentType represents the type of attachment.
type AttachmentType string

const (
	// AttachmentTypeFile is a file reference.
	AttachmentTypeFile AttachmentType = "file"
	// AttachmentTypeGitDiff is a git diff.
	AttachmentTypeGitDiff AttachmentType = "git_diff"
	// AttachmentTypeContext is contextual information.
	AttachmentTypeContext AttachmentType = "context"
	// AttachmentTypeScreenshot is a screenshot.
	AttachmentTypeScreenshot AttachmentType = "screenshot"
)
