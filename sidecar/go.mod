module github.com/argus-platform/argus/sidecar

go 1.24.0

require (
	github.com/argus-platform/argus/pkg v0.0.0
	go.uber.org/zap v1.27.0
)

require go.uber.org/multierr v1.11.0 // indirect

replace github.com/argus-platform/argus/pkg => ../pkg
