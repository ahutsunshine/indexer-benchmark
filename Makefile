.PHONY: vendor
vendor: export GOPRIVATE=github.com
vendor: # Ensures all go module dependencies are synced and copied to vendor
	@echo "Updating module dependencies..."
	@go mod tidy -v
	@go mod vendor -v
