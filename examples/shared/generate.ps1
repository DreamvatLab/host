protoc -I ./ --go_out=./ --go_opt=paths=source_relative ./test.proto
protoc -I ./ --go-grpc_out=require_unimplemented_servers=false:./ --go-grpc_opt=paths=source_relative ./test.proto
protoc-go-inject-tag -input="./test.pb.go"