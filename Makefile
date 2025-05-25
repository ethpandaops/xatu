proto:
	@echo "Buf generate:" ; \
	echo "-----------------" ; \
	for f in pkg/proto/*/; do \
		dir=$$(basename $$f); \
		if [[ $$dir == .* ]]; then \
			continue ; \
		fi ; \
		echo "	→ $$f" && buf generate --path "$$f"; \
	done ; \
	echo "-----------------" ;
