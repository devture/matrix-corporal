help: ## Show this help.
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'

_prepare_services: var/.env
	mkdir -p var/matrix-synapse-media-store var/matrix-synapse-postgres

var/.env:
	mkdir -p var
	echo 'CURRENT_USER_UID='`id -u` > var/.env;
	echo 'CURRENT_USER_GID='`id -g` >> var/.env

services-start: _prepare_services ## Starts all services (Postgres, Synapse, Riot)
	docker-compose --project-directory var -f etc/services/docker-compose.yaml -p matrix-corporal up -d

services-stop: _prepare_services ## Stops all services (Postgres, Synapse, Riot)
	docker-compose --project-directory var -f etc/services/docker-compose.yaml -p matrix-corporal down

services-tail-logs: _prepare_services ## Tails the logs for all running services
	docker-compose --project-directory var -f etc/services/docker-compose.yaml -p matrix-corporal logs -f

create-sample-system-user: _prepare_services ## Creates a system user, used for managing the Matrix server
	docker-compose --project-directory var -f etc/services/docker-compose.yaml -p matrix-corporal \
		exec synapse \
		register_new_matrix_user \
		-a \
		-u matrix-corporal \
		-p system-user-password \
		-c /data/homeserver.yaml \
		http://localhost:8008

run-postgres-cli: ## Starts a Postgres CLI (psql)
	docker-compose --project-directory var -f etc/services/docker-compose.yaml -p matrix-corporal \
		exec postgres \
		/bin/sh -c 'PGUSER=synapse PGPASSWORD=synapse-password PGDATABASE=homeserver psql -h postgres'

run-locally-quick: ## Builds and runs matrix-corporal locally (no containers, no govvv)
	go run matrix-corporal.go

run-locally: build-locally ## Builds and runs matrix-corporal locally (no containers)
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
	-p 41080:41080 \
	-p 41081:41081 \
	--mount type=bind,src=`pwd`/config.json,dst=/config.json,ro \
	--mount type=bind,src=`pwd`/policy.json,dst=/policy.json,ro \
	--network=matrix-corporal_default \
	devture/matrix-corporal:latest
