# AWS AMI Manager - IAM Permissions Documentation

This document outlines the required IAM permissions for aws-ami-manager to function properly across different operations and accounts.

## Overview

The tool requires different permissions depending on which operations you plan to use:
- **Copy**: List, describe, and copy AMIs; modify image attributes; create tags
- **Remove**: Deregister AMIs and delete snapshots
- **Cleanup**: List and describe AMIs; deregister and delete snapshots
- **Diagnose**: Read-only STS calls to verify credentials

## Default Account Permissions

The default account (where the tool is executed from) requires the following permissions:

### For Copy Operations

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeImages",
        "ec2:DescribeImageAttribute",
        "ec2:CopyImage",
        "ec2:ModifyImageAttribute",
        "ec2:CreateTags"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "sts:GetCallerIdentity"
      ],
      "Resource": "*"
    }
  ]
}
```

### For Remove Operations

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeImages",
        "ec2:DescribeImageAttribute",
        "ec2:DeregisterImage",
        "ec2:DeleteSnapshot"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "sts:GetCallerIdentity"
      ],
      "Resource": "*"
    }
  ]
}
```

### For Cleanup Operations

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeImages",
        "ec2:DescribeImageAttribute",
        "ec2:DeregisterImage",
        "ec2:DeleteSnapshot"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "sts:GetCallerIdentity"
      ],
      "Resource": "*"
    }
  ]
}
```

### Combined Policy (All Operations)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AmiManagement",
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeImages",
        "ec2:DescribeImageAttribute",
        "ec2:CopyImage",
        "ec2:DeregisterImage",
        "ec2:ModifyImageAttribute",
        "ec2:CreateTags",
        "ec2:DeleteSnapshot"
      ],
      "Resource": "*"
    },
    {
      "Sid": "STSIdentity",
      "Effect": "Allow",
      "Action": [
        "sts:GetCallerIdentity"
      ],
      "Resource": "*"
    }
  ]
}
```

## Cross-Account Permissions

When operating on target accounts (using `--accounts` and `--role` flags), each target account must have a role that:

1. Trusts the default account or the principal executing the tool
2. Has permissions for the operations being performed

### Target Account Role Trust Policy

Replace `111111111111` with your default/source account ID:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::111111111111:root"
      },
      "Action": "sts:AssumeRole",
      "Condition": {}
    }
  ]
}
```

Or, for more restricted access, specify the principal that will assume the role:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::111111111111:user/ami-manager"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
```

### Target Account Role Permissions

The assumed role in the target account needs the following policy attached:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AmiManagement",
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeImages",
        "ec2:DescribeImageAttribute",
        "ec2:ModifyImageAttribute",
        "ec2:CreateTags",
        "ec2:DeregisterImage",
        "ec2:DeleteSnapshot"
      ],
      "Resource": "*"
    }
  ]
}
```

## Required STS Permissions (Source Account)

To assume roles in target accounts, the source account needs:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AssumeRoleInTargetAccounts",
      "Effect": "Allow",
      "Action": "sts:AssumeRole",
      "Resource": [
        "arn:aws:iam::*:role/terraform",
        "arn:aws:iam::ACCOUNT_ID:role/YOUR_CUSTOM_ROLE_NAME"
      ]
    }
  ]
}
```

Replace `ACCOUNT_ID` with your target account IDs or use `*` to allow any account.

## Regional Considerations

All permissions above apply to all regions. If you want to restrict to specific regions, use Resource ARNs like:

```json
{
  "Effect": "Allow",
  "Action": ["ec2:*"],
  "Resource": "*",
  "Condition": {
    "StringEquals": {
      "aws:RequestedRegion": [
        "us-east-1",
        "eu-west-1",
        "ap-southeast-1"
      ]
    }
  }
}
```

## SSO Considerations

When using AWS SSO, the SSO role must have the above permissions. The role assumed by the tool (via `--role` or default) is in addition to the SSO role's permissions.

## Best Practices

1. **Use specific resource ARNs** when possible instead of `*`
2. **Apply least privilege** - only grant permissions for operations you need
3. **Use conditions** to restrict by region, account, or other criteria
4. **Audit regularly** - review what permissions are actually being used
5. **Use separate roles** for different operations if possible
6. **Enable CloudTrail** to audit all AMI operations

## Example: Restricted Copy-Only Role

For a read-only copy operation that only allows copying within a specific region:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "ReadSourceAmi",
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeImages",
        "ec2:DescribeImageAttribute"
      ],
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "aws:RequestedRegion": "us-east-1"
        }
      }
    },
    {
      "Sid": "CopyToTargetRegions",
      "Effect": "Allow",
      "Action": [
        "ec2:CopyImage",
        "ec2:ModifyImageAttribute",
        "ec2:CreateTags"
      ],
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "aws:RequestedRegion": [
            "eu-west-1",
            "ap-southeast-1"
          ]
        }
      }
    },
    {
      "Sid": "RequiredSTS",
      "Effect": "Allow",
      "Action": "sts:GetCallerIdentity",
      "Resource": "*"
    }
  ]
}
```

## Troubleshooting Permission Errors

If you encounter permission errors:

1. Run `aws-ami-manager diagnose --profile YOUR_PROFILE --loglevel=debug` to verify credentials
2. Check CloudTrail for the specific API calls that failed
3. Verify the role exists in the target account
4. Ensure the trust policy includes your principal
5. Confirm all required permissions are in the role policy

## References

- [AWS EC2 Permissions Reference](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-permissions.html)
- [AWS STS AssumeRole Documentation](https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html)
- [IAM Policy Simulator](https://policysim.aws.amazon.com/)

