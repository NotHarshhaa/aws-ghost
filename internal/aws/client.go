package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/rds"
)

// Client wraps AWS SDK clients
type Client struct {
	Config         aws.Config
	EC2            *ec2.Client
	ELB            *elasticloadbalancing.Client
	ELBv2          *elasticloadbalancingv2.Client
	RDS            *rds.Client
	ECR            *ecr.Client
	Lambda         *lambda.Client
	CloudWatch     *cloudwatch.Client
	CloudWatchLogs *cloudwatchlogs.Client
	AccountID      string
}

// NewClient creates a new AWS client with the given profile and region
func NewClient(profile, region string) (*Client, error) {
	var cfg aws.Config
	var err error

	opts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}

	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}

	cfg, err = config.LoadDefaultConfig(context.TODO(), opts...)
	if err != nil {
		return nil, err
	}

	// Get account ID
	stsSvc := NewSTSClient(cfg)
	accountID, err := stsSvc.GetAccountID()
	if err != nil {
		accountID = "unknown"
	}

	return &Client{
		Config:         cfg,
		EC2:            ec2.NewFromConfig(cfg),
		ELB:            elasticloadbalancing.NewFromConfig(cfg),
		ELBv2:          elasticloadbalancingv2.NewFromConfig(cfg),
		RDS:            rds.NewFromConfig(cfg),
		ECR:            ecr.NewFromConfig(cfg),
		Lambda:         lambda.NewFromConfig(cfg),
		CloudWatch:     cloudwatch.NewFromConfig(cfg),
		CloudWatchLogs: cloudwatchlogs.NewFromConfig(cfg),
		AccountID:      accountID,
	}, nil
}

// NewClientForRegion creates a new AWS client for a specific region
func NewClientForRegion(profile, region string) (*Client, error) {
	return NewClient(profile, region)
}
