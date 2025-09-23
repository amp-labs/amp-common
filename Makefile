.PHONY: fix
fix:
	wsl --allow-cuddle-declarations --fix ./... && \
		gci write . && \
		golangci-lint run -c .golangci.yml --fix

.PHONY: fix/sort
fix/sort:
	make fix | grep "" | sort

# Fix specific files passed as arguments
# Usage: make fix-files FILES="cmd/listen.go cmd/trigger.go"
fix-files:
	@if [ -z "$(FILES)" ]; then \
		echo "Usage: make fix-files FILES=\"file1.go file2.go ...\""; \
		echo "Example: make fix-files FILES=\"cmd/login.go cmd/logout.go\""; \
		exit 1; \
	fi
	@echo "Fixing files: $(FILES)"
	gci write $(FILES)
	@for file in $(FILES); do \
		echo "Formatting $$file..."; \
		gofmt -w $$file; \
	done
	@echo "Running go vet on packages containing the files..."
	@for file in $(FILES); do \
		dir=$$(dirname $$file); \
		echo "Vetting package ./$$dir/..."; \
		go vet ./$$dir/... || true; \
	done

.PHONY: test
test:
	go test -v ./...
