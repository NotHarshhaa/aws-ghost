package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/NotHarshhaa/aws-ghost/internal/aws"
	"github.com/NotHarshhaa/aws-ghost/internal/output"
	"github.com/NotHarshhaa/aws-ghost/internal/scanner"
	"github.com/NotHarshhaa/aws-ghost/internal/security"
	"github.com/NotHarshhaa/aws-ghost/pkg/types"
	"github.com/spf13/cobra"
)

var (
	region        string
	allRegions    bool
	profile       string
	only          string
	skip          string
	outputFmt     string
	minCost       float64
	idleDays      int
	noColor       bool
	quiet         bool
	securityLevel string
	validatePerms bool
	auditLog      bool
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan AWS account for ghost resources",
	Long:  `Scan your AWS account for forgotten, idle, and wasteful resources.`,
	RunE:  runScan,
}

func init() {
	scanCmd.Flags().StringVarP(&region, "region", "r", "us-east-1", "AWS region to scan")
	scanCmd.Flags().BoolVar(&allRegions, "all-regions", false, "Scan all enabled regions")
	scanCmd.Flags().StringVarP(&profile, "profile", "p", "", "AWS named profile")
	scanCmd.Flags().StringVar(&only, "only", "", "Comma-separated resource types to scan")
	scanCmd.Flags().StringVar(&skip, "skip", "", "Comma-separated resource types to skip")
	scanCmd.Flags().StringVarP(&outputFmt, "output", "o", "text", "Output format: text, json, markdown, csv")
	scanCmd.Flags().Float64Var(&minCost, "min-cost", 0, "Only show resources above this monthly cost ($)")
	scanCmd.Flags().IntVar(&idleDays, "idle-days", 7, "Days of inactivity to consider a resource idle")
	scanCmd.Flags().BoolVar(&noColor, "no-color", false, "Disable colored terminal output")
	scanCmd.Flags().BoolVar(&quiet, "quiet", false, "Only print the summary line")

	// Security flags
	scanCmd.Flags().StringVar(&securityLevel, "security-level", "medium", "Security level: low, medium, high, strict")
	scanCmd.Flags().BoolVar(&validatePerms, "validate-permissions", true, "Validate AWS permissions before scanning")
	scanCmd.Flags().BoolVar(&auditLog, "audit-log", true, "Enable security audit logging")
}

func runScan(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	// Parse include/exclude lists
	onlyList := parseList(only)
	skipList := parseList(skip)

	// Parse security level
	secLevel := types.SecurityLevel(securityLevel)
	secConfig := types.GetSecurityConfig(secLevel)
	secConfig.AuditLogging = auditLog
	secConfig.ValidatePermissions = validatePerms

	// Create AWS client
	client, err := aws.NewClient(profile, region)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Initialize security validator
	validator := security.NewValidator(secConfig, client.Config)

	// Validate credentials against security requirements
	credInfo, err := validator.ValidateCredentials(cmd.Context())
	if err != nil {
		validator.LogSecurityEvent(types.SecurityEvent{
			EventType: "credential_validation_failed",
			Message:   "Security validation failed",
			Reason:    err.Error(),
			Allowed:   false,
			User:      profile,
		})
		return fmt.Errorf("security validation failed: %w", err)
	}

	// Log successful credential validation
	validator.LogSecurityEvent(types.SecurityEvent{
		EventType: "credential_validation_success",
		Message:   "Security validation passed",
		Allowed:   true,
		User:      profile,
		AccountID: credInfo.AccountID,
	})

	// Create scanner registry
	registry := scanner.NewRegistry(client)

	// Get filtered scanners
	scanners := registry.GetFiltered(onlyList, skipList)

	// Scan
	var allResources []types.Resource
	var scannedTypes []string

	for name, scn := range scanners {
		// Validate resource access against security policy
		if err := validator.ValidateResourceAccess(name); err != nil {
			validator.LogSecurityEvent(types.SecurityEvent{
				EventType: "resource_access_denied",
				Message:   fmt.Sprintf("Access to resource type %s denied by security policy", name),
				Reason:    err.Error(),
				Allowed:   false,
				User:      profile,
				AccountID: credInfo.AccountID,
				Region:    region,
			})
			fmt.Fprintf(cmd.ErrOrStderr(), "Security: %s\n", err)
			continue
		}

		config := types.ScanConfig{
			Region:   region,
			Profile:  profile,
			IdleDays: idleDays,
			MinCost:  minCost,
		}

		// Validate scan configuration
		if err := validator.ValidateScanConfig(config); err != nil {
			validator.LogSecurityEvent(types.SecurityEvent{
				EventType: "scan_config_invalid",
				Message:   fmt.Sprintf("Invalid scan configuration for %s", name),
				Reason:    err.Error(),
				Allowed:   false,
				User:      profile,
			})
			fmt.Fprintf(cmd.ErrOrStderr(), "Security: %s\n", err)
			continue
		}

		resources, err := scn.Scan(config)
		if err != nil {
			validator.LogSecurityEvent(types.SecurityEvent{
				EventType: "scan_failed",
				Message:   fmt.Sprintf("Failed to scan %s", name),
				Reason:    err.Error(),
				Allowed:   false,
				User:      profile,
				AccountID: credInfo.AccountID,
				Region:    region,
			})
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to scan %s: %v\n", name, err)
			continue
		}

		// Filter by min cost
		var filtered []types.Resource
		for _, r := range resources {
			if r.MonthlyCost >= minCost {
				filtered = append(filtered, r)
			}
		}

		allResources = append(allResources, filtered...)
		scannedTypes = append(scannedTypes, name)
	}

	// Calculate total cost
	totalCost := 0.0
	for _, r := range allResources {
		totalCost += r.MonthlyCost
	}

	// Create result
	result := types.ScanResult{
		AccountID:    client.AccountID,
		Region:       region,
		Resources:    allResources,
		TotalCost:    totalCost,
		ScanDuration: time.Since(startTime),
		ScannedTypes: scannedTypes,
	}

	// Format output
	formatter, err := output.GetFormatter(outputFmt, noColor, quiet)
	if err != nil {
		return fmt.Errorf("failed to get formatter: %w", err)
	}

	outputStr, err := formatter.Format(result)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	// Write output
	if err := output.WriteOutput(outputStr, ""); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}

func parseList(input string) []string {
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
