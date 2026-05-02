package scanner

import (
	"context"
	"fmt"
	"time"

	"github.com/NotHarshhaa/aws-ghost/internal/aws"
	"github.com/NotHarshhaa/aws-ghost/internal/cost"
	"github.com/NotHarshhaa/aws-ghost/pkg/types"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// NATScanner scans for idle NAT Gateways
type NATScanner struct {
	client *aws.Client
	calc   *cost.Calculator
}

// NewNATScanner creates a new NAT Gateway scanner
func NewNATScanner(client *aws.Client) *NATScanner {
	return &NATScanner{
		client: client,
		calc:   cost.NewCalculator(),
	}
}

// Scan returns idle NAT Gateways
func (s *NATScanner) Scan(config types.ScanConfig) ([]types.Resource, error) {
	var resources []types.Resource

	resp, err := s.client.EC2.DescribeNatGateways(context.TODO(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to describe NAT gateways: %w", err)
	}

	for _, nat := range resp.NatGateways {
		// Skip deleting gateways
		if nat.State == "deleted" || nat.State == "deleting" {
			continue
		}

		// Check if NAT gateway has had zero traffic in the last 7 days
		if s.isIdleNATGateway(*nat.NatGatewayId, config.IdleDays) {
			idleDays := s.calculateIdleDays(nat.CreateTime)

			resource := types.Resource{
				ID:          *nat.NatGatewayId,
				Type:        "nat",
				Region:      s.client.Config.Region,
				Name:        getNATName(nat.Tags),
				State:       string(nat.State),
				IdleDays:    idleDays,
				MonthlyCost: s.calc.NATGatewayCost(),
				Metadata: map[string]string{
					"vpc_id":     *nat.VpcId,
					"subnet_id":  *nat.SubnetId,
					"created_at": nat.CreateTime.Format(time.RFC3339),
				},
				LastActive: *nat.CreateTime,
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

func (s *NATScanner) ResourceType() string {
	return "nat"
}

func (s *NATScanner) Description() string {
	return "Idle NAT Gateways"
}

func (s *NATScanner) isIdleNATGateway(natID string, idleDays int) bool {
	// For now, assume all NAT gateways older than idleDays are idle
	// In a full implementation, this would check CloudWatch metrics
	// for bytes processed over the time period
	return true
}

func (s *NATScanner) calculateIdleDays(createTime *time.Time) int {
	if createTime == nil {
		return 999
	}
	return int(time.Since(*createTime).Hours() / 24)
}

func getNATName(tags []ec2types.Tag) string {
	for _, tag := range tags {
		if tag.Key != nil && *tag.Key == "Name" {
			if tag.Value != nil {
				return *tag.Value
			}
		}
	}
	return ""
}
