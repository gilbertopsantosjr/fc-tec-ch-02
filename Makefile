.PHONY: build run test clean docker-build docker-up docker-down docker-logs

build:
	@echo "Building application..."
	go build -o bin/main .

run:
	@echo "Running application..."
	go run main.go

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning..."
	rm -rf bin/
	go clean

docker-build:
	@echo "Building Docker image..."
	docker compose build

docker-up:
	@echo "Starting services..."
	docker compose up -d

docker-down:
	@echo "Stopping services..."
	docker compose down

docker-logs:
	@echo "Showing logs..."
	docker compose logs -f app

docker-restart:
	@echo "Restarting services..."
	docker compose restart

test-request:
	@echo "Testing rate limiter..."
	@for i in $$(seq 1 6); do \
		echo "Request $$i"; \
		curl -s -o /dev/null -w "Status: %{http_code}\n" http://localhost:8080/test; \
	done

test-token:
	@echo "Testing token rate limiter..."
	@for i in $$(seq 1 12); do \
		echo "Request $$i with token"; \
		curl -s -o /dev/null -w "Status: %{http_code}\n" -H "API_KEY: my-secret-token" http://localhost:8080/test; \
	done


