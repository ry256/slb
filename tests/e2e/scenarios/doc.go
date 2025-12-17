// Package scenarios contains integration test scenarios for SLB workflows.
//
// Each file tests a specific workflow or feature:
//   - workflow_test.go: Full request→approve→execute workflows
//   - rejection_test.go: Request rejection scenarios
//   - timeout_test.go: Timeout and escalation behavior
//   - selfapproval_test.go: Self-approval prevention
package scenarios
