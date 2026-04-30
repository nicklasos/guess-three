.PHONY: run
run:
	go run ./main.go
	

.PHONY: dev
dev:
	@if [ -f .env ]; then set -a && source .env && set +a; fi && air
	
	
.PHONY: build
build:
	go build -o api ./main.go
	

.PHONY: restart
restart:
	sudo supervisorctl restart guess-three