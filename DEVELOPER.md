# AWS AMI Manager - Developer Quick Reference

## Quick Start for Development

### First Time Setup

```bash
# Clone the repository
git clone <repo-url>
cd aws-ami-manager

# Install dependencies
go mod download

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/tools/cmd/goimports@latest

# Build the application
make build

# Verify it works
./bin/aws-ami-manager --help
```

### Common Development Tasks

```bash
# Run all quality checks
make check

# Run tests only
make test

# View test coverage
make test-coverage
# Opens coverage.html in browser

# Format code
make fmt

# Check code quality without fixing
make lint
make vet

# Build binary
make build
```

### Code Quality Standards

Before committing:
```bash
# Run full check suite (will fail if issues found)
make check

# If check fails, fix issues then run again
make fmt    # Auto-format code
make check  # Re-run all checks
```

### Testing

```bash
# Run specific test
go test -v -run TestFormatTags ./aws

# Run with coverage
go test -v -cover ./aws

# Generate coverage report
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Adding New Tests

1. Create test functions following Go conventions: `TestFunctionName` in `*_test.go` files
2. Use table-driven tests for multiple scenarios:
   ```go
   tests := []struct {
       name     string
       input    string
       expected string
   }{
       {"case1", "input1", "expected1"},
       {"case2", "input2", "expected2"},
   }
   
   for _, tt := range tests {
       t.Run(tt.name, func(t *testing.T) {
           result := Function(tt.input)
           if result != tt.expected {
               t.Errorf("got %q, want %q", result, tt.expected)
           }
       })
   }
   ```

### Debugging

```bash
# Build with debug symbols
go build -o aws-ami-manager .

# Run with debug logging
./aws-ami-manager copy ... --loglevel=debug

# Run diagnose command to check configuration
./aws-ami-manager diagnose --profile YOUR_PROFILE --loglevel=debug
```

## Project Structure

```
aws-ami-manager/
├── main.go                 # Entry point
├── go.mod                  # Module definition
├── go.sum                  # Dependency checksums
├── Makefile               # Build automation
├── README.md              # User documentation
├── IAM_PERMISSIONS.md     # Permission reference
├── IMPROVEMENTS.md        # What was improved
│
├── cmd/                   # CLI commands
│   ├── root.go           # Root command
│   ├── copy.go           # Copy command
│   ├── remove.go         # Remove command
│   ├── cleanup.go        # Cleanup command
│   └── diagnose.go       # Diagnose command
│
└── aws/                   # AWS integration
    ├── ami.go            # AMI operations
    ├── ami_test.go       # AMI tests
    ├── config.go         # Configuration management
    ├── config_test.go    # Config tests
    └── credentials.go    # STS credentials
```

## Constants & Configuration

### DefaultAssumeRole

Location: `aws/config.go`

```go
const DefaultAssumeRole string = "terraform"
```

This is the default IAM role name assumed in cross-account operations. Override with:
- `--role` CLI flag
- `AWS_AMI_MANAGER_ROLE` environment variable

## Error Handling Patterns

### Error with Context
```go
if err != nil {
    return fmt.Errorf("operation_failed: %w", err)
}
```

### Error with Suggestions
```go
if credErr != nil {
    return nil, fmt.Errorf("credential failed: %v. Try: aws sso login --profile %s", credErr, profile)
}
```

## Logging Guidelines

Use logrus for all logging:

```go
import log "github.com/sirupsen/logrus"

// Info level - user-visible important info
log.Infof("AMI %s is available", amiID)

// Debug level - troubleshooting information
log.Debugf("Fetching metadata for %s", amiID)

// Warn level - something unexpected but not fatal
log.Warnf("AMI took %s to become available", elapsed)

// Error level - errors that don't cause fatal failure
log.Errorf("Failed to delete snapshot %s: %v", snapshotID, err)
```

## Linting Configuration

The project uses golangci-lint with custom configuration in `.golangci.yml`:

Key settings:
- Complexity limit: 15 (cyclomatic complexity)
- Enabled linters: vet, errcheck, staticcheck, unused, goimports, etc.
- Test files excluded from complexity checks

Run linting:
```bash
golangci-lint run --no-config ./...  # With go defaults
make lint                             # Using Makefile
```

## Common Issues & Solutions

### "golangci-lint: command not found"
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### "goimports: command not found"
```bash
go install golang.org/x/tools/cmd/goimports@latest
```

### Tests fail with credential errors
This is expected - tests need valid AWS credentials. Set up:
1. `aws sso login --profile YOUR_PROFILE`
2. Set `AWS_PROFILE=YOUR_PROFILE`
3. Or use static credentials in `~/.aws/credentials`

### Build fails with module issues
```bash
go mod tidy
go mod download
make build
```

## Git Workflow

Before pushing:
```bash
# Format code
make fmt

# Run quality checks
make check

# Build to verify
make build

# If all pass, commit and push
git add .
git commit -m "description"
git push
```

## Adding New Functions

1. Write the function in the appropriate package (aws/*, cmd/*)
2. Add godoc comment above the function if it's exported (starts with capital letter)
3. Add unit tests in `*_test.go` file
4. Run `make check` to verify
5. Update README if user-visible changes

Example function with godoc:
```go
// ProcessAmi performs the specified operation on the AMI.
// If dryRun is true, logs what would be done without making changes.
func (ami *Ami) ProcessAmi(operation string, dryRun bool) error {
    // implementation
}
```

## Performance Considerations

### Goroutines
The `Copy()` function uses goroutines for concurrent region operations:
```go
for region := range ami.AmisPerRegion {
    wg.Add(1)
    go func(region string) {
        // operation
        wg.Done()
    }(region)
}
wg.Wait()
```

### Timeouts
- AMI polling: 30-minute maximum, 5-minute warning threshold
- Backoff: 5s → 10s → 15s → 20s → 25s → 30s

### Memory
- EC2 clients are cached in `ec2Services` map by account and region
- Not cleared during operation, stays for connection reuse

## Documentation

Update these files when making changes:
- **README.md** - User-facing usage and installation
- **IAM_PERMISSIONS.md** - When permission requirements change
- **IMPROVEMENTS.md** - Summary of changes
- Godoc comments - For all public functions

---

For more information, see:
- README.md - User guide
- IAM_PERMISSIONS.md - Security and permissions
- IMPROVEMENTS.md - Recent changes and improvements

