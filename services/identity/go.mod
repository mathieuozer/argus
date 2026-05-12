module github.com/argus-platform/argus/services/identity

go 1.26.1

require (
	github.com/argus-platform/argus/pkg v0.0.0
	go.uber.org/zap v1.27.1
	google.golang.org/grpc v1.81.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/stretchr/testify v1.11.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
)

replace (
	github.com/argus-platform/argus/gen/go => ../../gen/go
	github.com/argus-platform/argus/pkg => ../../pkg
)
