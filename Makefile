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

run: build ## Builds and runs matrix-corporal
	./bin/matrix-corporal

build: ## Builds the matrix-corporal code
	gb build

test: ## Runs the tests
	gb test -v
