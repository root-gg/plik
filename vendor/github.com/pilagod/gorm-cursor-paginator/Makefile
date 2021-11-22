test:
	go test -v

test-env-up:
	docker-compose up -d

test-env-down:
	docker-compose down -v

test-coverage:
	go test --coverprofile=c.out
	go tool cover -html=c.out