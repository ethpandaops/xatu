proto:
	protoc --proto_path=./ --go_opt=module=github.com/ethpandaops/xatu/pkg/proto/eth/v1 --go_out=./pkg/proto/eth/v1/ pkg/proto/eth/v1/*.proto
	protoc --proto_path=./ --proto_path=./pkg/proto/eth/v1 --go_opt=module=github.com/ethpandaops/xatu/pkg/proto/xatu --go_out=./pkg/proto/xatu pkg/proto/xatu/*.proto