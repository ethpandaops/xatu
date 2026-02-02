proto:
	@echo "Buf generate:" ; \
	echo "-----------------" ; \
	for f in $$(find pkg/proto -type d); do \
		dir=$$(basename $$f); \
		if [[ $$dir == .* ]]; then \
			continue ; \
		fi ; \
		echo "	→ $$f" && buf generate --path "$$f"; \
	done ; \
	echo "-----------------" ;

# Sentry-logs development
.PHONY: sentry-logs-dev sentry-logs-build

sentry-logs-build:
	docker build -t ethpandaops/xatu-sentry-logs:local ./sentry-logs

sentry-logs-dev: sentry-logs-build
	@mkdir -p deploy/local/docker-compose/sentry-logs/logs
	docker compose up -d xatu-clickhouse-01 xatu-kafka xatu-server xatu-vector-http-kafka xatu-vector-kafka-clickhouse xatu-sentry-logs
	@echo ""
	@echo "Sentry-logs dev stack running."
	@echo ""
	@echo "Log file: deploy/local/docker-compose/sentry-logs/logs/geth.log"
	@echo "Vector logs: docker logs -f xatu-sentry-logs"
	@echo "Query: docker exec xatu-clickhouse-01 clickhouse-client --query 'SELECT * FROM default.execution_block_metrics'"
