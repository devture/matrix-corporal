GOLANG_CONTAINER_IMAGE := "docker.io/golang:1.25.0-alpine3.22"
GOLANGCI_LINT_CONTAINER_IMAGE := "docker.io/golangci/golangci-lint:v2.4.0"

help: ## Show this help.
	@grep -F -h "##" $(MAKEFILE_LIST) | grep -v grep | sed -e 's/\\$$//' | sed -e 's/##//'

_prepare_services: var/.env
	mkdir -p var/matrix-synapse-media-store var/matrix-synapse-postgres

var/.env:
	mkdir -p var
	echo 'CURRENT_USER_UID='`id -u` > var/.env;
	echo 'CURRENT_USER_GID='`id -g` >> var/.env

services-start: _prepare_services ## Starts all services (Postgres, Synapse, Element)
	docker compose --project-directory var --env-file var/.env -f etc/services/compose.yml -p matrix-corporal up -d

services-stop: _prepare_services ## Stops all services (Postgres, Synapse, Element)
	docker compose --project-directory var --env-file var/.env -f etc/services/compose.yml -p matrix-corporal down

services-tail-logs: _prepare_services ## Tails the logs for all running services
	docker compose --project-directory var --env-file var/.env -f etc/services/compose.yml -p matrix-corporal logs -f

create-sample-system-user: _prepare_services ## Creates a system user, used for managing the Matrix server
	docker compose --project-directory var --env-file var/.env -f etc/services/compose.yml -p matrix-corporal \
		exec synapse \
		register_new_matrix_user \
		-a \
		-u matrix-corporal \
		-p system-user-password \
		-c /data/homeserver.yaml \
		http://localhost:8008

run-postgres-cli: ## Starts a Postgres CLI (psql)
	docker compose --project-directory var --env-file var/.env -f etc/services/compose.yml -p matrix-corporal \
		exec postgres \
		/bin/sh -c 'PGUSER=synapse PGPASSWORD=synapse-password PGDATABASE=homeserver psql -h postgres'

run-locally-quick: ## Builds and runs matrix-corporal locally (no containers, no govvv)
	@echo "This doesn't work anymore."
	@echo ""
	@echo "Running matrix-corporal locally in this development environment means Synapse's REST auth provider won't be able to reach matrix-corporal."
	@echo "This will cause Interactive Auth to not function as expected."
	@echo ""
	@echo "Switch to 'make run-in-container-quick' for a fully working environment."
	@echo "Alternative, if you really insist on running locally, do: 'go run matrix-corporal.go'"
	@echo ""

	@exit 1

run-locally: build-locally ## Builds and runs matrix-corporal locally (no containers)
	@echo "Running locally is discouraged."
	@echo "Switch to 'make run-in-container-quick' for a fully working environment."

	./matrix-corporal

build-locally: ## Builds the matrix-corporal code locally (no containers)
	go get -u -v github.com/ahmetb/govvv
	rm -f matrix-corporal
	go build -a -ldflags "`~/go/bin/govvv -flags`" matrix-corporal.go

test: ## Runs the tests locally (no containers)
	go test ./...

build-container-image: ## Builds a Docker container image
	docker build -t devture/matrix-corporal:latest -f etc/docker/Dockerfile .

run-in-container: build-container-image ## Runs matrix-corporal in a container
	docker run \
	-it \
	--rm \
	--name=matrix-corporal \
	-p 127.0.0.1:41080:41080 \
	-p 127.0.0.1:41081:41081 \
	--mount type=bind,src=`pwd`/config.json,dst=/config.json,ro \
	--mount type=bind,src=`pwd`/policy.json,dst=/policy.json,ro \
	--network=matrix-corporal_default \
	devture/matrix-corporal:latest

run-in-container-quick: var/go ## Runs matrix-corporal in a container
	docker run \
	-it \
	--rm \
	--name=matrix-corporal \
	--user=`id -u`:`id -g` \
	--workdir=/work \
	-e GOPATH=/work/var/go/gopath \
	-e GOCACHE=/work/var/go/build-cache \
	-p 127.0.0.1:41080:41080 \
	-p 127.0.0.1:41081:41081 \
	--mount type=bind,src=`pwd`,dst=/work \
	--network=matrix-corporal_default \
	$(GOLANG_CONTAINER_IMAGE) \
	go run matrix-corporal.go

go-update-dependencies: var/go ## Updates all Go dependencies
	go get -u ./corporal/...
	go mod tidy -v

go-lint: var/go ## Runs golangci-lint
	docker run \
	--rm \
	-e GOPATH=/work/var/go/gopath \
	-e GOCACHE=/work/var/go/build-cache \
	-v $$(pwd):/work \
	-w /work \
	$(GOLANGCI_LINT_CONTAINER_IMAGE) \
	golangci-lint \
	run ./... -v

var/go:
	mkdir -p var/go/gopath 2>/dev/null
	mkdir -p var/go/build-cache 2>/dev/null
