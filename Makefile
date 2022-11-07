proto:
	protoc -I=./pkg/xatu/proto --go_out=. ./pkg/xatu/proto/decorated_event.proto