module github.com/azrsh/fragment-colocation-with-grpc

go 1.24.0

require (
	connectrpc.com/connect v1.18.1
	github.com/vektah/gqlparser/v2 v2.5.30
	google.golang.org/protobuf v1.36.6
)

require github.com/agnivade/levenshtein v1.2.1 // indirect

tool (
	connectrpc.com/connect/cmd/protoc-gen-connect-go
	google.golang.org/protobuf/cmd/protoc-gen-go
)
