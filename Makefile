.PHONY: clickhouse-routes
clickhouse-routes:
	@echo "Generating ClickHouse route code (requires Docker)..."
	go run ./pkg/clickhouse/route/cmd/generate

.PHONY: docker
# Build the local all-in-one image (xatu + cryo, for EL cannon). The cryo git
# ref is pinned in the Dockerfile (ARG CRYO_GIT_REF); override with
# --build-arg CRYO_GIT_REF=<ref> to test a different cryo build.
docker:
	docker build -t ethpandaops/xatu:local .

proto: .proto-clickhouse
	@echo "Buf generate:" ; \
	echo "-----------------" ; \
	for f in $$(find pkg/proto -type d); do \
		dir=$$(basename $$f); \
		if [[ $$dir == .* ]]; then \
			continue ; \
		fi ; \
		if [[ $$f == pkg/proto/clickhouse* ]]; then \
			continue ; \
		fi ; \
		echo "	→ $$f" && buf generate --path "$$f"; \
	done ; \
	echo "-----------------" ;

# Regenerates the typed ClickHouse read package (pkg/proto/clickhouse) from
# the migrations in deploy/migrations/clickhouse. Runs as part of make proto.
# Requires docker and buf.
.PHONY: .proto-clickhouse
.proto-clickhouse:
	docker compose up -d xatu-clickhouse-01 xatu-clickhouse-02 \
		xatu-clickhouse-zookeeper-01 xatu-clickhouse-zookeeper-02 xatu-clickhouse-zookeeper-03
	docker compose up xatu-clickhouse-migrator
	rm -f pkg/proto/clickhouse/*.go pkg/proto/clickhouse/*.proto
	rm -rf pkg/proto/clickhouse/clickhouse
	docker run --rm --network xatu_xatu-net -v "$$(pwd):/workspace" \
		ethpandaops/clickhouse-proto-gen:latest \
		--config /workspace/deploy/clickhouse-proto-gen.yaml \
		--dsn "clickhouse://xatu-clickhouse-01:9000/default" \
		--out /workspace/pkg/proto/clickhouse
	cd pkg/proto/clickhouse && buf generate
	go build ./pkg/proto/clickhouse/...

# Sentry-logs development
.PHONY: sentry-logs-dev sentry-logs-build

sentry-logs-build:
	docker build -t ethpandaops/xatu-sentry-logs:local ./sentry-logs

sentry-logs-dev: sentry-logs-build
	@mkdir -p deploy/local/docker-compose/sentry-logs/logs
	docker compose up -d xatu-clickhouse-01 xatu-kafka xatu-server xatu-consumoor xatu-sentry-logs
	@echo ""
	@echo "Sentry-logs dev stack running."
	@echo ""
	@echo "Log file: deploy/local/docker-compose/sentry-logs/logs/geth.log"
	@echo "Sentry logs: docker logs -f xatu-sentry-logs"
	@echo "Query: docker exec xatu-clickhouse-01 clickhouse-client --query 'SELECT * FROM default.execution_block_metrics'"
