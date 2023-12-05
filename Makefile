proto:
	protoc --proto_path=./ --go_opt=module=github.com/ethpandaops/xatu/pkg/proto/eth/v1 --go_out=./pkg/proto/eth/v1/ pkg/proto/eth/v1/*.proto
	protoc --proto_path=./ --go_opt=module=github.com/ethpandaops/xatu/pkg/proto/eth/v2 --go_out=./pkg/proto/eth/v2/ pkg/proto/eth/v2/*.proto
	protoc --proto_path=./ --proto_path=./pkg/proto/eth/v1 --proto_path=./pkg/proto/eth/v2 --go_opt=module=github.com/ethpandaops/xatu/pkg/proto/xatu --go-grpc_out=. --go-grpc_opt=paths=source_relative --go_out=./pkg/proto/xatu pkg/proto/xatu/*.proto
	protoc --proto_path=./ --go_opt=module=github.com/ethpandaops/xatu/pkg/proto/blockprint --go_out=./pkg/proto/blockprint pkg/proto/blockprint/*.proto
	protoc --proto_path=./ --go_opt=module=github.com/ethpandaops/xatu/pkg/proto/libp2p --go_out=./pkg/proto/libp2p pkg/proto/libp2p/*.proto