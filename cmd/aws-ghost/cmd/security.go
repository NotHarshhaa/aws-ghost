package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/NotHarshhaa/aws-ghost/pkg/types"
	"github.com/spf13/cobra"
)

var securityCmd = &cobra.Command{
	Use:   "security",
	Short: "Manage security settings and view security information",
	Long:  `Manage security settings, view current security configuration, and audit security events.`,
}

var securityInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show current security configuration",
	Long:  `Display the current security configuration and available security levels.`,
	RunE:  runSecurityInfo,
}

var securityLevelsCmd = &cobra.Command{
	Use:   "levels",
	Short: "Show available security levels",
	Long:  `Display all available security levels and their configurations.`,
	RunE:  runSecurityLevels,
}

func init() {
	rootCmd.AddCommand(securityCmd)
	securityCmd.AddCommand(securityInfoCmd)
	securityCmd.AddCommand(securityLevelsCmd)
}

func runSecurityInfo(cmd *cobra.Command, args []string) error {
	// Get current security level from flags or default
	level := types.SecurityLevel(securityLevel)
	if level == "" {
		level = types.SecurityLevelMedium
	}

	config := types.GetSecurityConfig(level)

	fmt.Printf("Current Security Configuration\n")
	fmt.Printf("=============================\n\n")
	fmt.Printf("Security Level: %s\n\n", string(level))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	
	fmt.Fprintf(w, "SETTING\tVALUE\tDESCRIPTION\n")
	fmt.Fprintf(w, "--------\t-----\t-----------\n")
	fmt.Fprintf(w, "MFA Required\t%t\tRequire Multi-Factor Authentication\n", config.RequireMFA)
	fmt.Fprintf(w, "Root Access\t%t\tAllow root account access\n", config.AllowRootAccess)
	fmt.Fprintf(w, "Max Idle Days\t%d\tMaximum days before resource is considered ghost\n", config.MaxIdleDays)
	fmt.Fprintf(w, "Audit Logging\t%t\tEnable security audit logging\n", config.AuditLogging)
	fmt.Fprintf(w, "Permission Validation\t%t\tValidate AWS permissions before scanning\n", config.ValidatePermissions)
	fmt.Fprintf(w, "Output Encryption\t%t\tEncrypt scan output\n", config.EncryptOutput)
	
	if len(config.AllowedRegions) > 0 {
		fmt.Fprintf(w, "Allowed Regions\t%v\tRegions where scanning is allowed\n", config.AllowedRegions)
	}
	
	if len(config.BlockedRegions) > 0 {
		fmt.Fprintf(w, "Blocked Regions\t%v\tRegions where scanning is blocked\n", config.BlockedRegions)
	}
	
	w.Flush()

	fmt.Printf("\nSecurity Recommendations:\n")
	fmt.Printf("- Use 'high' or 'strict' levels for production environments\n")
	fmt.Printf("- Enable MFA for all AWS accounts\n")
	fmt.Printf("- Regularly review audit logs for security events\n")
	fmt.Printf("- Consider restricting regions to those you actually use\n")

	return nil
}

func runSecurityLevels(cmd *cobra.Command, args []string) error {
	levels := []types.SecurityLevel{
		types.SecurityLevelLow,
		types.SecurityLevelMedium,
		types.SecurityLevelHigh,
		types.SecurityLevelStrict,
	}

	fmt.Printf("Available Security Levels\n")
	fmt.Printf("========================\n\n")

	for _, level := range levels {
		config := types.GetSecurityConfig(level)
		
		fmt.Printf("Level: %s\n", string(level))
		fmt.Printf("Description: %s\n", getLevelDescription(level))
		fmt.Printf("Best for: %s\n\n", getLevelUseCase(level))
		
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "  SETTING\tVALUE\n")
		fmt.Fprintf(w, "  --------\t-----\n")
		fmt.Fprintf(w, "  MFA Required\t%t\n", config.RequireMFA)
		fmt.Fprintf(w, "  Root Access\t%t\n", config.AllowRootAccess)
		fmt.Fprintf(w, "  Max Idle Days\t%d\n", config.MaxIdleDays)
		fmt.Fprintf(w, "  Audit Logging\t%t\n", config.AuditLogging)
		fmt.Fprintf(w, "  Permission Validation\t%t\n", config.ValidatePermissions)
		fmt.Fprintf(w, "  Output Encryption\t%t\n", config.EncryptOutput)
		w.Flush()
		
		if len(config.AllowedRegions) > 0 {
			fmt.Printf("  Allowed Regions\t%v\n", config.AllowedRegions)
		}
		
		fmt.Printf("\n%s\n\n", getLevelWarning(level))
	}

	fmt.Printf("Usage Examples:\n")
	fmt.Printf("  aws-ghost scan --security-level low\n")
	fmt.Printf("  aws-ghost scan --security-level high --validate-permissions\n")
	fmt.Printf("  aws-ghost scan --security-level strict --audit-log\n")

	return nil
}

func getLevelDescription(level types.SecurityLevel) string {
	switch level {
	case types.SecurityLevelLow:
		return "Minimal security restrictions for development/testing"
	case types.SecurityLevelMedium:
		return "Balanced security for general use"
	case types.SecurityLevelHigh:
		return "Enhanced security for production environments"
	case types.SecurityLevelStrict:
		return "Maximum security with comprehensive restrictions"
	default:
		return "Unknown level"
	}
}

func getLevelUseCase(level types.SecurityLevel) string {
	switch level {
	case types.SecurityLevelLow:
		return "Development, testing, personal projects"
	case types.SecurityLevelMedium:
		return "General scanning, non-production environments"
	case types.SecurityLevelHigh:
		return "Production environments, compliance requirements"
	case types.SecurityLevelStrict:
		return "Highly regulated environments, maximum security"
	default:
		return "Unknown use case"
	}
}

func getLevelWarning(level types.SecurityLevel) string {
	switch level {
	case types.SecurityLevelLow:
		return "⚠️  Low security level allows root access and has minimal restrictions"
	case types.SecurityLevelMedium:
		return "✅ Recommended for most use cases with good security balance"
	case types.SecurityLevelHigh:
		return "🔒 High security level requires MFA and has stricter controls"
	case types.SecurityLevelStrict:
		return "🛡️  Strict mode severely limits operations and regions"
	default:
		return ""
	}
}
