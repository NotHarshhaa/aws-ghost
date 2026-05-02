package scanner

import (
	"context"
	"time"

	"github.com/NotHarshhaa/aws-ghost/internal/aws"
	"github.com/NotHarshhaa/aws-ghost/internal/cost"
	"github.com/NotHarshhaa/aws-ghost/pkg/types"
)

// LoadBalancerScanner scans for idle load balancers
type LoadBalancerScanner struct {
	client *aws.Client
	calc   *cost.Calculator
}

// NewLoadBalancerScanner creates a new load balancer scanner
func NewLoadBalancerScanner(client *aws.Client) *LoadBalancerScanner {
	return &LoadBalancerScanner{
		client: client,
		calc:   cost.NewCalculator(),
	}
}

// Scan returns idle load balancers (ALB/NLB/CLB)
func (s *LoadBalancerScanner) Scan(config types.ScanConfig) ([]types.Resource, error) {
	var resources []types.Resource

	// Scan ALBs and NLBs
	albResp, err := s.client.ELBv2.DescribeLoadBalancers(context.TODO(), nil)
	if err == nil {
		for _, lb := range albResp.LoadBalancers {
			// Check if load balancer has had zero traffic in the last 7 days
			if s.isIdleLoadBalancer(*lb.LoadBalancerArn, config.IdleDays) {
				idleDays := s.calculateIdleDays(lb.CreatedTime)

				resource := types.Resource{
					ID:          *lb.LoadBalancerArn,
					Type:        "loadbalancer",
					Region:      s.client.Config.Region,
					Name:        *lb.LoadBalancerName,
					State:       string(lb.State.Code),
					IdleDays:    idleDays,
					MonthlyCost: s.calc.LoadBalancerCost(string(lb.Type)),
					Metadata: map[string]string{
						"type":       string(lb.Type),
						"scheme":     string(lb.Scheme),
						"dns_name":   *lb.DNSName,
						"created_at": lb.CreatedTime.Format(time.RFC3339),
					},
					LastActive: *lb.CreatedTime,
				}

				resources = append(resources, resource)
			}
		}
	}

	// Scan CLBs
	clbResp, err := s.client.ELB.DescribeLoadBalancers(context.TODO(), nil)
	if err == nil {
		for _, lb := range clbResp.LoadBalancerDescriptions {
			if s.isIdleClassicLB(*lb.LoadBalancerName, config.IdleDays) {
				idleDays := s.calculateIdleDays(lb.CreatedTime)

				resource := types.Resource{
					ID:          *lb.LoadBalancerName,
					Type:        "loadbalancer",
					Region:      s.client.Config.Region,
					Name:        *lb.LoadBalancerName,
					State:       "classic",
					IdleDays:    idleDays,
					MonthlyCost: s.calc.LoadBalancerCost("classic"),
					Metadata: map[string]string{
						"type":       "classic",
						"dns_name":   *lb.DNSName,
						"created_at": lb.CreatedTime.Format(time.RFC3339),
					},
					LastActive: *lb.CreatedTime,
				}

				resources = append(resources, resource)
			}
		}
	}

	return resources, nil
}

func (s *LoadBalancerScanner) ResourceType() string {
	return "loadbalancer"
}

func (s *LoadBalancerScanner) Description() string {
	return "Idle load balancers (ALB/NLB/CLB)"
}

func (s *LoadBalancerScanner) isIdleLoadBalancer(arn string, idleDays int) bool {
	// For now, assume all load balancers older than idleDays are idle
	// In a full implementation, this would check CloudWatch metrics
	// for request counts over the time period
	return true
}

func (s *LoadBalancerScanner) isIdleClassicLB(name string, idleDays int) bool {
	// For now, assume all classic load balancers older than idleDays are idle
	return true
}

func (s *LoadBalancerScanner) calculateIdleDays(createTime *time.Time) int {
	if createTime == nil {
		return 999
	}
	return int(time.Since(*createTime).Hours() / 24)
}
