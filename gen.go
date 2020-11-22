package hiprost

//go:generate protoc --go_out=paths=source_relative:. hiprost.proto
//go:generate protoc --go-grpc_out=paths=source_relative:. hiprost.proto
