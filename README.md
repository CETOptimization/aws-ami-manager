# AWS Ami Manager

Aws-ami-manager offers a simple way to perform copy, remove and cleanup operations on your AMI's. 

PS: Forked from original repo thus not using CETOptimization related builds.

## Usage

This application uses the typical ways of authenticating with AWS, including:
- Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN, AWS_REGION)
- Shared config/credentials files with profiles (`--profile` or `AWS_PROFILE`)
- AWS SSO profiles (run `aws sso login --profile <name>` first)
- Instance / pod role credentials (IMDS / IRSA)

A helper command is available to inspect what the tool resolves:
```
./aws-ami-manager diagnose --region eu-west-1 --profile my-sso-admin --loglevel=debug
```

### Copy
```
./aws-ami-manager copy \
  --amiID=ami-0e94877fc6310ea8b \
  --regions=eu-west-1,eu-central-1 \
  --accounts=123456789012,987654321098 \
  --role CrossAccountAmiRole \
  --region eu-west-1
```
Copies the AMI from the default region (resolved from profile / env / `--region`) to the list of specified regions and grants launch permissions to the listed accounts by assuming the provided role in each account.

### Remove
Remove an AMI in the current (default) account:
```
./aws-ami-manager remove --amiID ami-0123456789abcdef0 --region eu-west-1
```
Assume a role into another account before removing:
```
./aws-ami-manager remove \
  --amiID ami-0123456789abcdef0 \
  --accounts 222222222222 \
  --role TerraformDeploymentRole \
  --region eu-west-1
```
Dry run (no changes) – shows what would be deleted including snapshots:
```
./aws-ami-manager remove \
  --amiID ami-0123456789abcdef0 \
  --accounts 222222222222 \
  --role TerraformDeploymentRole \
  --region eu-west-1 \
  --dry-run
```

### Cleanup
(Existing behavior) Keeps the newest AMIs matching specific tag filters per region and removes older ones.

### Diagnose
Use this to debug credential/region issues:
```
./aws-ami-manager diagnose --profile my-sso-admin --region eu-west-1 --loglevel=debug
```
It prints detected profile, region, and attempts to fetch the STS caller identity.

## Common SSO Notes
If using AWS SSO:
1. Define an SSO profile in `~/.aws/config` with `sso_start_url`, `sso_region`, `sso_account_id`, `sso_role_name`, and `region`.
2. Run `aws sso login --profile <profileName>`.
3. Pass `--profile <profileName>` (or export `AWS_PROFILE`).
4. Run a quick test: `aws sts get-caller-identity` should succeed before using the tool.

## Example Assume Role Trust Policy (Target Account Role)
```
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": { "AWS": "arn:aws:iam::111111111111:root" },
      "Action": "sts:AssumeRole"
    }
  ]
}
```
Grant the base (SSO) role permission to assume the target role by attaching a policy with `sts:AssumeRole` on the target role ARN(s).

## Flags Overview
- `--region` Override or set the AWS region.
- `--profile` Specify a shared config profile.
- `--accounts` (copy/remove) Account IDs for permissioning or assumption (remove uses only the first right now).
- `--role` IAM role name to assume in target accounts.
- `--dry-run` (remove) Preview deregistration and snapshot removal.
- `--loglevel` debug|info|warn|error.

## Licence
Apache License, version 2.0

## Build
GitHub Actions is used to build the application now.

Local manual build:
```
go build -o dist/aws-ami-manager .
```
