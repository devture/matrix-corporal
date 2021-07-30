# Development / Experimenting

If you'd like to contribute code to this project or give it a try locally (before deploying it), you need to:

- clone this repository

- get [Docker](https://www.docker.com/) and [Docker Compose](https://docs.docker.com/compose/) -- used for running a local Matrix Synapse + element-web setup, for testing

- start all dependency services (Postgres, Matrix Synapse, element-web): `make services-start`. You can stop them later with `make services-stop` or tail their logs with `make services-tail-logs`

- create a sample "system" user: `make create-sample-system-user`

- copy the sample configuration: `cp config.json.dist config.json`

- copy the sample policy: `cp policy.json.dist policy.json`

- build and run the `matrix-corporal` program by executing: `make run-in-container-quick`

- you should now be able to log in with user `a` and password `test` (as per the policy) to the [element-web instance](http://matrix-corporal.127.0.0.1.nip.io:41465)

- you should also be able to log in with the system user `matrix-corporal` and password `system-user-password` to the [element-web instance](http://matrix-corporal.127.0.0.1.nip.io:41465)

- create a few rooms or communities manually, through element-web with that system (`matrix-corporal`) user

- modify `policy.json` (e.g. defining new managed rooms/communities, definining users, defining community/room memberships, etc) and watch `matrix-corporal` reconciliate the server state

For local development, it's best to install a [Go](https://golang.org/) compiler (version 1.12 or later is required) locally.
Some tests are available and can be executed with: `make test`.
