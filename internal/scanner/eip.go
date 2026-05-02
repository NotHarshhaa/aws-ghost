package scanner

import (
	"context"
	"fmt"
	"time"

	"github.com/NotHarshhaa/aws-ghost/internal/aws"
	"github.com/NotHarshhaa/aws-ghost/internal/cost"
	"github.com/NotHarshhaa/aws-ghost/pkg/types"
)

// EIPScanner scans for unattached Elastic IPs
type EIPScanner struct {
	client *aws.Client
	calc   *cost.Calculator
}

// NewEIPScanner creates a new EIP scanner
func NewEIPScanner(client *aws.Client) *EIPScanner {
	return &EIPScanner{
		client: client,
		calc:   cost.NewCalculator(),
	}
}

// Scan returns unattached Elastic IPs
func (s *EIPScanner) Scan(config types.ScanConfig) ([]types.Resource, error) {
	var resources []types.Resource

	resp, err := s.client.EC2.DescribeAddresses(context.TODO(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to describe addresses: %w", err)
	}

	for _, addr := range resp.Addresses {
		// Check if IP is not associated with a running instance
		if addr.AssociationId == nil || addr.InstanceId == nil {
			// Use a default idle time since AllocationTime is not available
			idleDays := 30

			resource := types.Resource{
				ID:          *addr.AllocationId,
				Type:        "eip",
				Region:      s.client.Config.Region,
				Name:        *addr.PublicIp,
				State:       "unattached",
				IdleDays:    idleDays,
				MonthlyCost: s.calc.ElasticIPCost(),
				Metadata: map[string]string{
					"public_ip": *addr.PublicIp,
					"domain":    string(addr.Domain),
				},
				LastActive: time.Now().AddDate(0, 0, -idleDays),
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

func (s *EIPScanner) ResourceType() string {
	return "eip"
}

func (s *EIPScanner) Description() string {
	return "Unattached Elastic IPs"
}
