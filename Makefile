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
