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

// EBSScanner scans for unattached EBS volumes
type EBSScanner struct {
	client *aws.Client
	calc   *cost.Calculator
}

// NewEBSScanner creates a new EBS scanner
func NewEBSScanner(client *aws.Client) *EBSScanner {
	return &EBSScanner{
		client: client,
		calc:   cost.NewCalculator(),
	}
}

// Scan returns unattached EBS volumes
func (s *EBSScanner) Scan(config types.ScanConfig) ([]types.Resource, error) {
	var resources []types.Resource

	resp, err := s.client.EC2.DescribeVolumes(context.TODO(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to describe volumes: %w", err)
	}

	for _, vol := range resp.Volumes {
		// Check if volume is unattached
		if len(vol.Attachments) == 0 && vol.State != "deleting" && vol.State != "deleted" {
			idleDays := s.calculateIdleDays(vol.CreateTime)

			size := int32(0)
			if vol.Size != nil {
				size = *vol.Size
			}
			resource := types.Resource{
				ID:          *vol.VolumeId,
				Type:        "ebs",
				Region:      s.client.Config.Region,
				Name:        getVolumeName(vol.Tags),
				State:       string(vol.State),
				IdleDays:    idleDays,
				MonthlyCost: s.calc.EBSVolumeCost(int(size), string(vol.VolumeType)),
				Metadata: map[string]string{
					"size":       fmt.Sprintf("%d GB", size),
					"type":       string(vol.VolumeType),
					"encrypted":  fmt.Sprintf("%t", vol.Encrypted),
					"created_at": vol.CreateTime.Format(time.RFC3339),
				},
				LastActive: *vol.CreateTime,
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

func (s *EBSScanner) ResourceType() string {
	return "ebs"
}

func (s *EBSScanner) Description() string {
	return "Unattached EBS volumes"
}

func (s *EBSScanner) calculateIdleDays(createTime *time.Time) int {
	if createTime == nil {
		return 999
	}
	return int(time.Since(*createTime).Hours() / 24)
}

func getVolumeName(tags []ec2types.Tag) string {
	for _, tag := range tags {
		if tag.Key != nil && *tag.Key == "Name" {
			if tag.Value != nil {
				return *tag.Value
			}
		}
	}
	return ""
}
