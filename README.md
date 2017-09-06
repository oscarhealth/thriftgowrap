This repo is a proof of concept for generating client wrappers for thrift in Golang.


To see Thrift Go Wrapper in action, assuming your CWD is the README's location and thriftgowrap is in a src folder in your $GOPATH.

1. Generate thrift service to relative thriftgowrap/generated/services directory: `thrift -out .. --gen go thrift/multiplication.thrift`. 
2. To Generate wrapped client at thriftgowrap/generated/client/multiplication `go generate thriftgowrap/generated/...`