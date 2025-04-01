module github.com/tencent-lke/lke-sdk-go

go 1.23.5

require github.com/r3labs/sse/v2 v2.10.0

require (
	github.com/google/uuid v1.6.0 // indirect
	gopkg.in/cenkalti/backoff.v1 v1.1.0 // indirect
)

require (
	github.com/json-iterator/go v1.1.12
	github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/openai/openai-go v0.1.0-beta.3
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	golang.org/x/net v0.34.0 // indirect
)

replace github.com/r3labs/sse/v2 => github.com/TeCHiScy/sse/v2 v2.11.0
