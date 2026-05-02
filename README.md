# aws-ghost

> **Scan your AWS account for forgotten, idle, and wasteful resources — and see exactly what they're costing you.**

Most AWS bills have ghosts in them. Unattached EBS volumes quietly billing at $0.10/GB/month. Elastic IPs reserved but attached to nothing. NAT Gateways in dead VPCs. Snapshots from instances that were terminated a year ago. `aws-ghost` finds all of them in one shot.

```bash
$ aws-ghost scan --region us-east-1

 AWS Ghost Scanner ────────────────────────────────────────────
  Account   123456789012
  Region    us-east-1
  Scanned   14 resource types in 12s

 Ghosts Found: 23 resources — estimated waste: $284.50/month

 EBS Volumes (unattached)                          $109.20/mo
  vol-0a1b2c3d4e   500 GB  gp2   idle 47 days     $50.00/mo
  vol-0f9e8d7c6b   300 GB  gp3   idle 112 days    $36.00/mo
  vol-0123456789   230 GB  gp2   idle 8 days      $23.20/mo

 Elastic IPs (unattached)                          $10.80/mo
  52.14.xxx.xxx    us-east-1a   idle 203 days      $3.60/mo
  18.232.xxx.xxx   us-east-1b   idle 67 days       $3.60/mo
  34.201.xxx.xxx   us-east-1c   idle 31 days       $3.60/mo

 Load Balancers (zero traffic, 7d)                 $48.50/mo
  alb-staging-old   0 req/day   last active 34d    $16.20/mo
  nlb-test-infra    0 req/day   last active 89d    $16.20/mo
  alb-demo-env      0 req/day   last active 14d    $16.10/mo

 NAT Gateways (zero traffic, 7d)                   $97.20/mo
  nat-0a1b2c3d      vpc-legacy   0 bytes/day       $32.40/mo
  nat-0e5f6a7b      vpc-staging  0 bytes/day       $32.40/mo
  nat-0c9d8e7f      vpc-sandbox  0 bytes/day       $32.40/mo

 RDS Snapshots (older than 90 days)                $18.80/mo
  prod-db-snap-2024-01-14    210 GB   311 days old  $6.30/mo
  staging-snap-2023-12-01    180 GB   354 days old  $5.40/mo
  test-db-final-snap         240 GB   290 days old  $7.10/mo

 Estimated annual savings if cleaned: $3,414.00

 Run `aws-ghost report --output markdown > ghost-report.md` to export
──────────────────────────────────────────────────────────────────
```

---

## Why this exists

AWS makes it very easy to create resources and very easy to forget them. `aws-nuke` is a wrecking ball — it deletes everything. `aws-ghost` is the opposite: **read-only, safe, and honest**. It tells you what's wasting money so *you* can decide what to do with it.

| | aws-ghost | aws-nuke | AWS Cost Explorer |
|---|---|---|---|
| Read-only (safe) | ✓ | ✗ | ✓ |
| Per-resource waste estimate | ✓ | ✗ | ✗ |
| Idle detection (not just unattached) | ✓ | ✗ | ✗ |
| CLI / terminal output | ✓ | ✓ | ✗ |
| No SaaS / no account needed | ✓ | ✓ | ✗ |
| Multi-account support | ✓ | ✓ | ✗ |

---

## Install

### Homebrew (macOS / Linux)
```bash
brew install NotHarshhaa/tap/aws-ghost
```

### Go install
```bash
go install github.com/NotHarshhaa/aws-ghost@latest
```

### Binary
```bash
curl -sSL https://github.com/NotHarshhaa/aws-ghost/releases/latest/download/aws-ghost_linux_amd64 \
  -o /usr/local/bin/aws-ghost && chmod +x /usr/local/bin/aws-ghost
```

### Docker
```bash
docker run --rm \
  -e AWS_ACCESS_KEY_ID \
  -e AWS_SECRET_ACCESS_KEY \
  -e AWS_DEFAULT_REGION \
  ghcr.io/NotHarshhaa/aws-ghost scan
```

---

## Usage

`aws-ghost` uses your existing AWS credentials — same as the AWS CLI (`~/.aws/credentials`, env vars, or IAM role).

### Scan everything
```bash
aws-ghost scan
```

### Scan a specific region
```bash
aws-ghost scan --region ap-south-1
```

### Scan all regions at once
```bash
aws-ghost scan --all-regions
```

### Scan specific resource types only
```bash
aws-ghost scan --only ebs,eip,snapshots
```

### Multi-account via AWS profiles
```bash
aws-ghost scan --profile prod
aws-ghost scan --profile staging
```

### Output formats
```bash
# Default pretty terminal output
aws-ghost scan

# JSON — pipe to jq, feed into scripts
aws-ghost scan --output json | jq '.resources[] | select(.monthly_cost > 20)'

# Markdown — paste into Notion, Confluence, incident doc
aws-ghost scan --output markdown > ghost-report.md

# CSV — open in Excel, share with finance team
aws-ghost scan --output csv > waste-report.csv
```

### Cost threshold filter
```bash
# Only show resources wasting more than $10/month
aws-ghost scan --min-cost 10
```

### Idle threshold
```bash
# Flag resources idle for more than 30 days (default: 7)
aws-ghost scan --idle-days 30
```

---

## What it scans

| Resource | Ghost condition | Cost estimate |
|---|---|---|
| EBS Volumes | Unattached for N+ days | $/GB/month by volume type |
| Elastic IPs | Not associated with a running instance | $0.005/hr flat |
| Load Balancers (ALB/NLB/CLB) | Zero requests in last 7 days | LCU + hourly rate |
| NAT Gateways | Zero bytes processed in last 7 days | $0.045/hr + data |
| RDS Snapshots | Older than 90 days, no retention policy | $/GB/month |
| EC2 Snapshots | Older than 90 days, source volume gone | $/GB/month |
| ECR Images | Untagged or last pulled 90+ days ago | $/GB/month |
| Lambda Functions | Zero invocations in last 30 days | negligible, flagged |
| CloudWatch Log Groups | No retention policy set, large size | $/GB ingested |
| Unused Security Groups | Not attached to any resource | informational |
| Empty Target Groups | Registered but no healthy targets | informational |
| Idle EC2 Instances | CPU < 5% avg over 14 days | full instance cost |
| Stopped EC2 Instances | Stopped for 14+ days (EBS still billing) | EBS cost |
| Old AMIs | Not used by any running instance, 90+ days old | snapshot cost |

---

## Flags

| Flag | Description | Default |
|---|---|---|
| `--region` | AWS region to scan | `us-east-1` |
| `--all-regions` | Scan all enabled regions | `false` |
| `--profile` | AWS named profile | default credential chain |
| `--only` | Comma-separated resource types to scan | all |
| `--skip` | Comma-separated resource types to skip | none |
| `--output` | Output format: `text`, `json`, `markdown`, `csv` | `text` |
| `--min-cost` | Only show resources above this monthly cost ($) | `0` |
| `--idle-days` | Days of inactivity to consider a resource idle | `7` |
| `--no-color` | Disable colored terminal output | `false` |
| `--quiet` | Only print the summary line | `false` |

---

## CI/CD integration

Run `aws-ghost` as a scheduled CI job to catch waste before it compounds.

### GitHub Actions
```yaml
name: AWS Ghost Scan
on:
  schedule:
    - cron: '0 9 * * 1'  # Every Monday at 9am

jobs:
  ghost-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install aws-ghost
        run: go install github.com/NotHarshhaa/aws-ghost@latest

      - name: Scan for ghost resources
        env:
          AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
          AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          AWS_DEFAULT_REGION: us-east-1
        run: aws-ghost scan --output markdown --min-cost 5 > ghost-report.md

      - name: Upload report
        uses: actions/upload-artifact@v4
        with:
          name: ghost-report
          path: ghost-report.md
```

### GitLab CI
```yaml
aws-ghost-scan:
  image: golang:1.21
  script:
    - go install github.com/NotHarshhaa/aws-ghost@latest
    - aws-ghost scan --all-regions --output markdown > ghost-report.md
  artifacts:
    paths:
      - ghost-report.md
  only:
    - schedules
```

---

## Security & Trust

`aws-ghost` is designed with security and transparency as core principles. Your AWS credentials are never sent to any external service.

### 🔒 Security Guarantees

- **Local-only credential usage**: Your AWS credentials are used locally by the tool and never transmitted to any external servers
- **Read-only operations**: The tool only uses AWS Describe/List API calls — it never creates, modifies, or deletes resources
- **API call transparency**: Every AWS API call made during a scan is logged and displayed in the summary
- **Open source**: The entire codebase is open source and auditable by anyone
- **No account required**: You don't need to create an account or sign up to use this tool

### 🛡️ Security Features

#### Credential Verification
Before each scan, `aws-ghost` displays:
- Your AWS Account ID
- User ARN being used
- Credential source (environment variables, AWS profile, IAM role, etc.)
- Whether root account access is being used (with warning if true)
- MFA status verification

#### Read-Only Verification
The tool verifies that all API operations are read-only:
- Automatic detection of any write operations (Create, Delete, Update, etc.)
- Warning if any non-read operations are detected
- Summary of all services and operations used during the scan

#### Security Audit Command
Run a comprehensive security audit of your credentials:

```bash
aws-ghost security audit
```

This will:
- Verify your credential configuration
- Check for root account usage
- Verify MFA status
- Provide security recommendations
- Display required IAM permissions

#### Security Levels
Configure security strictness with four levels:

```bash
aws-ghost scan --security-level low      # Minimal restrictions (dev/testing)
aws-ghost scan --security-level medium   # Balanced (default)
aws-ghost scan --security-level high     # Enhanced (production)
aws-ghost scan --security-level strict   # Maximum restrictions
```

View available security levels:
```bash
aws-ghost security levels
```

---

## Required IAM permissions

`aws-ghost` is **read-only**. It will never modify or delete anything.

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:Describe*",
        "elasticloadbalancing:Describe*",
        "rds:Describe*",
        "ecr:Describe*",
        "ecr:ListImages",
        "lambda:List*",
        "lambda:GetFunction",
        "logs:Describe*",
        "cloudwatch:GetMetricStatistics",
        "cloudwatch:ListMetrics"
      ],
      "Resource": "*"
    }
  ]
}
```

---

## Roadmap

- [ ] `--fix` flag — interactive prompt to delete confirmed ghosts one by one
- [ ] Slack / Teams webhook: post weekly ghost report automatically
- [ ] Tag-based filtering: skip resources tagged `keep=true` or `env=prod`
- [ ] Terraform output: generate a `terraform destroy` plan for ghost resources
- [ ] AWS Organizations support: scan all accounts in an org
- [ ] Cost trend: compare this week's ghosts vs last week

---

## Contributing

```bash
git clone https://github.com/NotHarshhaa/aws-ghost
cd aws-ghost
go mod tidy
go run . scan --region us-east-1
```

Issues and PRs are welcome. If you find a ghost resource type that's not covered, open an issue with the resource type and how to detect it.

---

## License

MIT © [NotHarshhaa](https://github.com/NotHarshhaa)
