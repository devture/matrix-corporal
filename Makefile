help: ## Show this help.
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'

services-start: ## Starts all services (Postgres, Synapse, Riot)
	mkdir -p var/matrix-synapse-media-store;
	chmod 777 var/matrix-synapse-media-store;
	docker-compose -f etc/services/docker-compose.yaml -p matrix-corporal up -d

services-stop: ## Stops all services (Postgres, Synapse, Riot)
	docker-compose -f etc/services/docker-compose.yaml -p matrix-corporal down

services-tail-logs: ## Tails the logs for all running services
	docker-compose -f etc/services/docker-compose.yaml -p matrix-corporal logs -f

create-sample-system-user: ## Creates a system user, used for managing the Matrix server
	docker-compose -f etc/services/docker-compose.yaml -p matrix-corporal \
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
