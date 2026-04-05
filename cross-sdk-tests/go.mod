module github.com/agent-receipts/ar/cross-sdk-tests

go 1.26.1

replace github.com/agent-receipts/ar/sdk/go => ../sdk/go

require github.com/agent-receipts/ar/sdk/go v0.0.0-00010101000000-000000000000

require github.com/google/uuid v1.6.0 // indirect
