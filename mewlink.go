package mewlink

//go:generate go install -v google.golang.org/protobuf/cmd/protoc-gen-go@latest
//go:generate protoc --go_out=. --go_opt=paths=import --go_opt=module=github.com/AsenHu/mewlink ./protos/v1/roominfo.proto
