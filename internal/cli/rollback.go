package cli

import (
	"fmt"
	"time"

	"github.com/Dicklesworthstone/slb/internal/db"
	"github.com/Dicklesworthstone/slb/internal/output"
	"github.com/spf13/cobra"
)

var (
	flagRollbackForce bool
)

func init() {
	rollbackCmd.Flags().BoolVarP(&flagRollbackForce, "force", "f", false, "force rollback even if state may be stale")

	rootCmd.AddCommand(rollbackCmd)
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback <request-id>",
	Short: "Rollback an executed command",
	Long: `Rollback the effects of an executed command using captured state.

Rollback requires that:
1. The request was executed (status: executed or execution_failed)
2. Rollback state was captured before execution (--capture-rollback flag)
3. The captured state is still valid

Note: Not all commands can be rolled back. Rollback is only available when
pre-execution state capture was enabled.

Examples:
  slb rollback abc123
  slb rollback abc123 --force`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		requestID := args[0]

		// Open database
		dbConn, err := db.Open(GetDB())
		if err != nil {
			return fmt.Errorf("opening database: %w", err)
		}
		defer dbConn.Close()

		// Get the request
		request, err := dbConn.GetRequest(requestID)
		if err != nil {
			return fmt.Errorf("getting request: %w", err)
		}

		// Validate request state
		if request.Status != db.StatusExecuted && request.Status != db.StatusExecutionFailed {
			return fmt.Errorf("cannot rollback: request status is %s (must be executed or execution_failed)", request.Status)
		}

		// Check for rollback data
		if request.Rollback == nil || request.Rollback.Path == "" {
			return fmt.Errorf("no rollback data available for this request (was --capture-rollback used?)")
		}

		// Check if already rolled back
		if request.Rollback.RolledBackAt != nil {
			if !flagRollbackForce {
				return fmt.Errorf("request was already rolled back at %s (use --force to rollback again)",
					request.Rollback.RolledBackAt.Format(time.RFC3339))
			}
		}

		// TODO: Implement actual rollback logic
		// This would depend on what kind of state was captured:
		// - File backups: restore from backup path
		// - Git state: reset to captured commit
		// - Database snapshot: restore from dump
		// For now, we just record the rollback attempt

		// Build output
		type rollbackResult struct {
			RequestID    string `json:"request_id"`
			RollbackPath string `json:"rollback_path"`
			RolledBackAt string `json:"rolled_back_at"`
			Status       string `json:"status"`
			Message      string `json:"message"`
		}

		now := time.Now().UTC()
		resp := rollbackResult{
			RequestID:    requestID,
			RollbackPath: request.Rollback.Path,
			RolledBackAt: now.Format(time.RFC3339),
			Status:       "pending",
			Message:      "Rollback functionality is not yet implemented. Rollback data is available at the specified path.",
		}

		out := output.New(output.Format(GetOutput()))
		if GetOutput() == "json" {
			return out.Write(resp)
		}

		// Human-readable output
		fmt.Printf("Rollback for request %s\n", requestID)
		fmt.Printf("Rollback data: %s\n", request.Rollback.Path)
		fmt.Println()
		fmt.Println("Note: Automatic rollback is not yet implemented.")
		fmt.Println("You can manually restore from the captured state at the path above.")

		return nil
	},
}
