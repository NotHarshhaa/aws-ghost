package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var (
	scheduleCron    string
	scheduleCommand string
	scheduleProfile string
	scheduleRegion  string
	scheduleList    bool
	scheduleRemove  string
	scheduleEnable  bool
	scheduleDisable bool
)

var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Manage scheduled automated scans",
	Long:  `Configure automated scheduled scans with built-in cron functionality.`,
	RunE:  runSchedule,
}

func init() {
	scheduleCmd.Flags().StringVar(&scheduleCron, "cron", "0 9 * * 1", "Cron expression (default: every Monday at 9am)")
	scheduleCmd.Flags().StringVar(&scheduleCommand, "command", "scan", "Command to run (scan, check, anomaly)")
	scheduleCmd.Flags().StringVar(&scheduleProfile, "profile", "", "AWS profile to use")
	scheduleCmd.Flags().StringVar(&scheduleRegion, "region", "us-east-1", "AWS region to scan")
	scheduleCmd.Flags().BoolVar(&scheduleList, "list", false, "List all scheduled scans")
	scheduleCmd.Flags().StringVar(&scheduleRemove, "remove", "", "Remove a scheduled scan by ID")
	scheduleCmd.Flags().BoolVar(&scheduleEnable, "enable", false, "Enable a scheduled scan")
	scheduleCmd.Flags().BoolVar(&scheduleDisable, "disable", false, "Disable a scheduled scan")
}

func runSchedule(cmd *cobra.Command, args []string) error {
	if scheduleList {
		return listSchedules()
	}
	if scheduleRemove != "" {
		return removeSchedule(scheduleRemove)
	}
	if scheduleEnable {
		return toggleSchedule(args[0], true)
	}
	if scheduleDisable {
		return toggleSchedule(args[0], false)
	}

	// Add new schedule
	return addSchedule()
}

func addSchedule() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	ghostDir := filepath.Join(configDir, "aws-ghost")
	if err := os.MkdirAll(ghostDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	schedulePath := filepath.Join(ghostDir, "schedules.json")

	// Load existing schedules
	var schedules []ScheduledScan
	data, err := os.ReadFile(schedulePath)
	if err == nil {
		json.Unmarshal(data, &schedules)
	}

	// Create new schedule
	newSchedule := ScheduledScan{
		ID:        generateScheduleID(),
		Cron:      scheduleCron,
		Command:   scheduleCommand,
		Profile:   scheduleProfile,
		Region:    scheduleRegion,
		Enabled:   true,
		CreatedAt: time.Now().Format(time.RFC3339),
		NextRun:   calculateNextRun(scheduleCron),
	}

	schedules = append(schedules, newSchedule)

	// Save schedules
	data, err = json.MarshalIndent(schedules, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal schedules: %w", err)
	}

	if err := os.WriteFile(schedulePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write schedules: %w", err)
	}

	fmt.Printf("✅ Schedule added successfully\n")
	fmt.Printf("\nSchedule Details:\n")
	fmt.Printf("  ID: %s\n", newSchedule.ID)
	fmt.Printf("  Cron: %s\n", newSchedule.Cron)
	fmt.Printf("  Command: %s\n", newSchedule.Command)
	fmt.Printf("  Profile: %s\n", newSchedule.Profile)
	fmt.Printf("  Region: %s\n", newSchedule.Region)
	fmt.Printf("  Next Run: %s\n", newSchedule.NextRun)
	fmt.Printf("\nNote: To actually run scheduled scans, set up a system cron job or use a task scheduler:\n")
	fmt.Printf("  Linux/Mac: Add to crontab: %s aws-ghost schedule run %s\n", newSchedule.Cron, newSchedule.ID)
	fmt.Printf("  Windows: Use Task Scheduler to run: aws-ghost schedule run %s\n", newSchedule.ID)

	return nil
}

func listSchedules() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	schedulePath := filepath.Join(configDir, "aws-ghost", "schedules.json")

	data, err := os.ReadFile(schedulePath)
	if err != nil {
		fmt.Println("No scheduled scans configured.")
		fmt.Println("\nTo add a schedule:")
		fmt.Println("  aws-ghost schedule --cron '0 9 * * 1' --command scan")
		return nil
	}

	var schedules []ScheduledScan
	if err := json.Unmarshal(data, &schedules); err != nil {
		return fmt.Errorf("failed to parse schedules: %w", err)
	}

	fmt.Printf("\n📅 Scheduled Scans\n")
	fmt.Printf("==================\n\n")

	if len(schedules) == 0 {
		fmt.Println("No scheduled scans configured.")
		return nil
	}

	for i, s := range schedules {
		status := "✅ Enabled"
		if !s.Enabled {
			status = "❌ Disabled"
		}
		fmt.Printf("%d. %s\n", i+1, s.ID)
		fmt.Printf("   Status: %s\n", status)
		fmt.Printf("   Cron: %s\n", s.Cron)
		fmt.Printf("   Command: %s\n", s.Command)
		fmt.Printf("   Profile: %s\n", s.Profile)
		fmt.Printf("   Region: %s\n", s.Region)
		fmt.Printf("   Next Run: %s\n", s.NextRun)
		fmt.Printf("   Created: %s\n", s.CreatedAt)
		fmt.Printf("\n")
	}

	fmt.Println("Commands:")
	fmt.Println("  aws-ghost schedule --remove <id>    Remove a schedule")
	fmt.Println("  aws-ghost schedule --enable <id>    Enable a schedule")
	fmt.Println("  aws-ghost schedule --disable <id>   Disable a schedule")
	fmt.Println("  aws-ghost schedule run <id>         Run a schedule manually")

	return nil
}

func removeSchedule(id string) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	schedulePath := filepath.Join(configDir, "aws-ghost", "schedules.json")

	data, err := os.ReadFile(schedulePath)
	if err != nil {
		return fmt.Errorf("no schedules found: %w", err)
	}

	var schedules []ScheduledScan
	if err := json.Unmarshal(data, &schedules); err != nil {
		return fmt.Errorf("failed to parse schedules: %w", err)
	}

	// Find and remove the schedule
	var updated []ScheduledScan
	found := false
	for _, s := range schedules {
		if s.ID == id {
			found = true
			continue
		}
		updated = append(updated, s)
	}

	if !found {
		return fmt.Errorf("schedule with ID %s not found", id)
	}

	// Save updated schedules
	data, err = json.MarshalIndent(updated, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal schedules: %w", err)
	}

	if err := os.WriteFile(schedulePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write schedules: %w", err)
	}

	fmt.Printf("✅ Schedule %s removed successfully\n", id)
	return nil
}

func toggleSchedule(id string, enable bool) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	schedulePath := filepath.Join(configDir, "aws-ghost", "schedules.json")

	data, err := os.ReadFile(schedulePath)
	if err != nil {
		return fmt.Errorf("no schedules found: %w", err)
	}

	var schedules []ScheduledScan
	if err := json.Unmarshal(data, &schedules); err != nil {
		return fmt.Errorf("failed to parse schedules: %w", err)
	}

	// Find and toggle the schedule
	found := false
	for i := range schedules {
		if schedules[i].ID == id {
			schedules[i].Enabled = enable
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("schedule with ID %s not found", id)
	}

	// Save updated schedules
	data, err = json.MarshalIndent(schedules, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal schedules: %w", err)
	}

	if err := os.WriteFile(schedulePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write schedules: %w", err)
	}

	action := "enabled"
	if !enable {
		action = "disabled"
	}
	fmt.Printf("✅ Schedule %s %s successfully\n", id, action)
	return nil
}

func generateScheduleID() string {
	return fmt.Sprintf("sched-%d", time.Now().Unix())
}

func calculateNextRun(cronExpr string) string {
	// Simplified next run calculation
	// In a real implementation, use a proper cron library
	return "Next execution based on cron: " + cronExpr
}

type ScheduledScan struct {
	ID        string `json:"id"`
	Cron      string `json:"cron"`
	Command   string `json:"command"`
	Profile   string `json:"profile"`
	Region    string `json:"region"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
	NextRun   string `json:"next_run"`
}
