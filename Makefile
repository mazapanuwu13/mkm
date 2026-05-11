.PHONY: build run test lint clean info watch-logs

# Build the mkm binary
build:
	go build -o mkm .

# Run the TUI application
run:
	./mkm

# Run all unit tests
test:
	go test ./... -v

# Run the Go linter
lint:
	go vet ./...

# Remove build artifacts
clean:
	rm -f mkm

# Print system and project info
info:
	@echo "=== AutoHost TUI - Project Info ==="
	@echo "Go version : $$(go version)"
	@echo "Module     : $$(head -1 go.mod)"
	@echo "Directory  : $$(pwd)"
	@echo "Files      : $$(find . -name '*.go' | wc -l) Go files"
	@echo "==================================="

# Simulate a long-running process (streaming output demo)
watch-logs:
	@echo "[autohost] Starting log stream..."
	@for i in 1 2 3 4 5; do \
		echo "[autohost] Service heartbeat #$$i — $$(date)"; \
		sleep 1; \
	done
	@echo "[autohost] Done."
