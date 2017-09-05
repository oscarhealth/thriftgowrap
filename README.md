To see in action assuming your CWD is the README's location.

1. Generate thrift service to relative hisocar/generated/services directory: `thrift -out .. --gen go thrift/multiplication.thrift`. 
2. To Generate wrapped client at hioscar/generated/client/multiplication `go generate hisocar/generated/...`