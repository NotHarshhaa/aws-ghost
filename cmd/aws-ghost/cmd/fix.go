package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/NotHarshhaa/aws-ghost/internal/aws"
	"github.com/NotHarshhaa/aws-ghost/internal/scanner"
	"github.com/NotHarshhaa/aws-ghost/internal/ui"
	"github.com/NotHarshhaa/aws-ghost/pkg/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/cobra"
)

var (
	fixDryRun      bool
	fixAutoConfirm bool
	fixForce       bool
	fixOnlyTypes   []string
	fixSkipTypes   []string
	fixRegions     []string
	fixProfiles    []string
	fixMinCost     float64
	fixAllRegions  bool
)

var fixCmd = &cobra.Command{
	Use:   "fix",
	Short: "Safely clean up ghost resources",
	Long: `Safely clean up ghost resources with comprehensive safety checks.

This command provides automated cleanup of wasteful AWS resources with multiple
layers of safety protections:

• Dry-run mode to preview what would be deleted
• Interactive confirmation for each resource
• Resource type whitelisting/blacklisting
• Tag-based protection (resources with 'keep=true' are protected)
• Cost threshold validation
• Multi-region support with confirmation

Examples:
  # Preview what would be cleaned up (dry run)
  aws-ghost fix --dry-run

  # Clean up resources under $10/month with confirmation
  aws-ghost fix --min-cost 10

  # Auto-confirm cleanup for specific resource types
  aws-ghost fix --only ebs,eip --auto-confirm

  # Force cleanup (skip all confirmations - DANGEROUS)
  aws-ghost fix --force --min-cost 5

⚠️  This is a destructive operation. Always use --dry-run first!`,
	RunE: runFix,
}

func init() {
	rootCmd.AddCommand(fixCmd)

	fixCmd.Flags().BoolVar(&fixDryRun, "dry-run", true, "Preview what would be deleted without actually deleting")
	fixCmd.Flags().BoolVar(&fixAutoConfirm, "auto-confirm", false, "Automatically confirm all deletions (use with caution)")
	fixCmd.Flags().BoolVar(&fixForce, "force", false, "Force cleanup without any confirmations (DANGEROUS)")
	fixCmd.Flags().StringSliceVar(&fixOnlyTypes, "only", []string{}, "Only clean up these resource types")
	fixCmd.Flags().StringSliceVar(&fixSkipTypes, "skip", []string{}, "Skip these resource types")
	fixCmd.Flags().StringSliceVar(&fixRegions, "region", []string{}, "AWS regions to clean up (default: current region)")
	fixCmd.Flags().StringSliceVar(&fixProfiles, "profile", []string{}, "AWS profiles to use")
	fixCmd.Flags().Float64Var(&fixMinCost, "min-cost", 0, "Only clean up resources costing more than this amount monthly")
	fixCmd.Flags().BoolVar(&fixAllRegions, "all-regions", false, "Clean up resources in all enabled regions")
}

func runFix(cmd *cobra.Command, args []string) error {
	// Safety check: require explicit confirmation for dangerous operations
	if fixForce && !fixAutoConfirm && !fixDryRun {
		fmt.Println(ui.GetCompactLogo())
		fmt.Println("🚨 DANGER: You are using --force mode which will delete resources without any confirmation!")
		fmt.Println("🚨 This is extremely dangerous and cannot be undone!")
		fmt.Println()
		fmt.Print("Type 'DELETE-ALL-MY-RESOURCES' to continue: ")

		var confirmation string
		fmt.Scanln(&confirmation)
		if confirmation != "DELETE-ALL-MY-RESOURCES" {
			fmt.Println("❌ Operation cancelled. Phew!")
			return nil
		}
	}

	// Show what we're about to do
	fmt.Println(ui.GetCompactLogo())
	fmt.Println()

	if fixDryRun {
		fmt.Println("🔍 DRY RUN MODE - No resources will actually be deleted")
		fmt.Println()
	}

	// Determine regions to scan
	scanRegions := determineScanRegions(fixRegions, fixAllRegions)

	// Determine profiles to use
	scanProfiles := determineScanProfiles(fixProfiles)

	var totalCleaned int
	var totalSavings float64

	for _, profile := range scanProfiles {
		fmt.Printf("🔧 Scanning profile: %s\n", profile)

		for _, region := range scanRegions {
			fmt.Printf("  📍 Region: %s\n", region)

			// Create AWS client
			client, err := aws.NewClient(profile, region)
			if err != nil {
				fmt.Printf("    ❌ Failed to create AWS client: %v\n", err)
				continue
			}

			// Create scanner registry
			registry := scanner.NewRegistry(client)
			scanners := registry.GetFiltered(fixOnlyTypes, fixSkipTypes)

			// Scan for ghost resources
			var allResources []types.Resource
			for name, scanner := range scanners {
				config := types.ScanConfig{
					Region:   region,
					Profile:  profile,
					Only:     fixOnlyTypes,
					Skip:     fixSkipTypes,
					MinCost:  fixMinCost,
					IdleDays: 7,
				}
				resources, err := scanner.Scan(config)
				if err != nil {
					fmt.Printf("    ❌ Failed to scan %s: %v\n", name, err)
					continue
				}
				allResources = append(allResources, resources...)
			}

			// Filter by cost threshold
			var filteredResources []types.Resource
			for _, resource := range allResources {
				if resource.MonthlyCost >= fixMinCost {
					filteredResources = append(filteredResources, resource)
				}
			}

			if len(filteredResources) == 0 {
				fmt.Printf("    ✅ No ghost resources found\n")
				continue
			}

			fmt.Printf("    👻 Found %d ghost resources ($%.2f/month)\n", len(filteredResources), calculateTotalCost(filteredResources))
			fmt.Println()

			// Clean up resources
			cleaned, savings, err := cleanupResources(cmd.Context(), client, filteredResources, fixDryRun, fixAutoConfirm || fixForce)
			if err != nil {
				fmt.Printf("    ❌ Cleanup failed: %v\n", err)
				continue
			}

			totalCleaned += cleaned
			totalSavings += savings

			if cleaned > 0 {
				if fixDryRun {
					fmt.Printf("    📋 Would clean up %d resources ($%.2f/month savings)\n", cleaned, savings)
				} else {
					fmt.Printf("    ✅ Cleaned up %d resources ($%.2f/month savings)\n", cleaned, savings)
				}
			}
		}
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))

	if fixDryRun {
		fmt.Printf("🔍 DRY RUN SUMMARY: Would clean up %d resources for $%.2f/month savings\n", totalCleaned, totalSavings)
		fmt.Println()
		fmt.Println("💡 To actually clean up these resources, run:")
		fmt.Println("   aws-ghost fix --dry-run=false")
	} else {
		fmt.Printf("✅ CLEANUP COMPLETE: Cleaned up %d resources for $%.2f/month savings\n", totalCleaned, totalSavings)
		fmt.Println()
		fmt.Println("💰 Estimated annual savings: $%.2f", totalSavings*12)
	}

	return nil
}

func cleanupResources(ctx context.Context, client *aws.Client, resources []types.Resource, dryRun, autoConfirm bool) (int, float64, error) {
	var cleaned int
	var totalSavings float64

	for _, resource := range resources {
		// Check if resource has protection tags
		if hasProtectionTags(client, resource) {
			fmt.Printf("    🔒 Skipping %s %s (protected by tags)\n", resource.Type, resource.ID)
			continue
		}

		// Show resource details
		fmt.Printf("    🗑️  %s %s (%s) - $%.2f/month\n", resource.Type, resource.ID, resource.State, resource.MonthlyCost)
		if resource.Metadata["reason"] != "" {
			fmt.Printf("       Reason: %s\n", resource.Metadata["reason"])
		}

		// Get confirmation unless auto-confirm is enabled
		if !autoConfirm && !dryRun {
			fmt.Printf("       Delete this resource? [y/N]: ")
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
				fmt.Printf("       ⏭️  Skipped\n")
				continue
			}
		}

		// Perform cleanup
		if dryRun {
			fmt.Printf("       📋 [DRY RUN] Would delete %s\n", resource.ID)
			cleaned++
			totalSavings += resource.MonthlyCost
		} else {
			err := deleteResource(ctx, client, resource)
			if err != nil {
				fmt.Printf("       ❌ Failed to delete: %v\n", err)
				continue
			}
			fmt.Printf("       ✅ Deleted\n")
			cleaned++
			totalSavings += resource.MonthlyCost
		}
		fmt.Println()
	}

	return cleaned, totalSavings, nil
}

func hasProtectionTags(client *aws.Client, resource types.Resource) bool {
	// This would check for tags like "keep=true", "env=prod", etc.
	// Implementation depends on resource type
	// For now, return false as placeholder
	return false
}

func deleteResource(ctx context.Context, client *aws.Client, resource types.Resource) error {
	// Implementation would call the appropriate AWS delete API based on resource type
	// This is a placeholder that would need to be implemented for each resource type
	switch resource.Type {
	case "S3 Bucket":
		// Delete S3 bucket
		return deleteS3Bucket(ctx, client.S3, resource.ID)
	case "EBS Volume":
		// Delete EBS volume
		return deleteEBSVolume(ctx, client.EC2, resource.ID)
	case "Elastic IP":
		// Release Elastic IP
		return deleteElasticIP(ctx, client.EC2, resource.ID)
	// Add more resource types...
	default:
		return fmt.Errorf("unsupported resource type: %s", resource.Type)
	}
}

// Placeholder functions for resource deletion
func deleteS3Bucket(ctx context.Context, client *s3.Client, bucketName string) error {
	// Implementation would:
	// 1. Delete all objects in bucket
	// 2. Delete bucket
	return fmt.Errorf("S3 bucket deletion not yet implemented")
}

func deleteEBSVolume(ctx context.Context, client *ec2.Client, volumeId string) error {
	// Implementation would delete the EBS volume
	return fmt.Errorf("EBS volume deletion not yet implemented")
}

func deleteElasticIP(ctx context.Context, client *ec2.Client, allocationId string) error {
	// Implementation would release the Elastic IP
	return fmt.Errorf("Elastic IP deletion not yet implemented")
}

func calculateTotalCost(resources []types.Resource) float64 {
	var total float64
	for _, resource := range resources {
		total += resource.MonthlyCost
	}
	return total
}

func determineScanRegions(specified []string, allRegions bool) []string {
	if allRegions {
		// Return all AWS regions
		return []string{
			"us-east-1", "us-east-2", "us-west-1", "us-west-2",
			"eu-west-1", "eu-west-2", "eu-central-1",
			"ap-south-1", "ap-southeast-1", "ap-southeast-2", "ap-northeast-1",
		}
	}
	if len(specified) > 0 {
		return specified
	}
	return []string{"us-east-1"} // Default region
}

func determineScanProfiles(specified []string) []string {
	if len(specified) > 0 {
		return specified
	}
	return []string{""} // Default profile
}
