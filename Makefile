.PHONY: dev dev-otel

dev:
	@trap 'kill 0' INT; \
	(cd backend && go run ./cmd/main.go) & \
	(cd frontend && pnpm dev) & \
	wait

dev-otel:
	@trap 'kill 0' INT; \
	(cd backend && ENABLE_OTEL=true go run ./cmd/main.go) & \
	(cd frontend && pnpm dev) & \
	wait
