# Justfile for flexpond-version-generator

set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

# Default task
default: build

# Build the CLI
build:
	go build -v ./...

# Install the CLI locally (GOBIN or GOPATH/bin)
install:
	go install .

# Run the CLI (accepts additional args)
run *ARGS:
	go run . {{ARGS}}

# Run unit tests
test:
	go test ./...

# Tidy modules
tidy:
	go mod tidy

# Upgrade all dependencies to latest minor/patch and tidy
upgrade:
	go get -u ./...
	go mod tidy

# Lint (basic)
vet:
	go vet ./...
