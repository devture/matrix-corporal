# Development / Experimenting

If you'd like to contribute code to this project or give it a try locally (before deploying it), you need to:

- clone this repository

- get [Docker](https://www.docker.com/) and [Docker Compose](https://docs.docker.com/compose/) -- used for running a local Matrix Synapse + riot-web setup, for testing

- get a [Go](https://golang.org/) compiler (version 1.12 or later is required)

- start all dependency services (Postgres, Matrix Synapse, riot-web): `make services-start`. You can stop them later with `make services-stop` or tail their logs with `make services-tail-logs`

- create a sample "system" user: `make create-sample-system-user`

- copy the sample configuration: `cp config.json.dist config.json`

- copy the sample policy: `cp policy.json.dist policy.json`

- build and run the `matrix-corporal` program: `make run`

- you should now be able to log in with user `a` and password `test` (as per the policy) to the [riot-web instance](http://matrix-corporal.127.0.0.1.xip.io:41465)

- you should also be able to log in with the system user `matrix-corporal` and password `system-user-password` to the [riot-web instance](http://matrix-corporal.127.0.0.1.xip.io:41465)

- create a few rooms or communities manually, through riot-web with that system (`matrix-corporal`) user

- modify `policy.json` (e.g. defining new managed rooms/communities, definining users, defining community/room memberships, etc) and watch `matrix-corporal` reconciliate the server state

Some tests are available and can be executed with: `make test`.
