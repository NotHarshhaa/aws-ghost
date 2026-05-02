package scanner

import (
	"github.com/NotHarshhaa/aws-ghost/internal/aws"
	"github.com/NotHarshhaa/aws-ghost/internal/cost"
	"github.com/NotHarshhaa/aws-ghost/pkg/types"
)

// ECRScanner scans for unused ECR images
type ECRScanner struct {
	client *aws.Client
	calc   *cost.Calculator
}

// NewECRScanner creates a new ECR scanner
func NewECRScanner(client *aws.Client) *ECRScanner {
	return &ECRScanner{
		client: client,
		calc:   cost.NewCalculator(),
	}
}

// Scan returns unused ECR images
func (s *ECRScanner) Scan(config types.ScanConfig) ([]types.Resource, error) {
	// TODO: Implement ECR scanner - temporarily disabled due to type issues
	return []types.Resource{}, nil
}

func (s *ECRScanner) ResourceType() string {
	return "ecr"
}

func (s *ECRScanner) Description() string {
	return "Unused ECR images"
}
