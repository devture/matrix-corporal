# Matrix Corporal Setup

Setting up `matrix-corporal` can vary depending on:

- how you've installed Matrix Synapse (manually, in Docker containers, etc.)

- what kind of reverse proxy you have in front of it (none, nginx, Apache, other)

- what other services ([mxisd](https://github.com/kamax-io/mxisd), etc) and modules (password providers, etc.) you're running

It's not possible to provide setup steps that suit your exact case, but this document will attempt to give a guide, so you can adapt things to your needs.

The easiest way to install `matrix-corporal` is by installing all Matrix-related services (Synapse and others) using the [matrix-docker-ansible-deploy](https://github.com/spantaleev/matrix-docker-ansible-deploy) Ansible playbook. The playbook supports installing and integrating `matrix-corporal` into all Matrix services.

For all other setup cases, see below.


## Installing Matrix Corporal

Building the program can be done manually (see the [development](development.md) guide).

Alternatively, you can pull the [devture/matrix-corporal](https://hub.docker.com/r/devture/matrix-corporal) Docker image.


## Configuring Matrix Corporal

You can refer to the [configuration](configuration.md) document to learn about configuring `matrix-corporal`.

Most of the work you've got to do is figuring out which [policy provider](policy-providers.md) to use.

A lot of the other values that go into the configuration file are either shared secrets (which you can generate with a command like `pwgen -s 128 1` or other) or shared secrets coming frmo Matrix Synapse's configuration (`homeserver.yaml`).


## Matrix Synapse configuration

You need to set up the [Shared Secret Authenticator](https://github.com/devture/matrix-synapse-shared-secret-auth) password provider module for Matrix Synapse. This is necessary, so that the `matrix-corporal` user (defined in `Corporal.UserId` in the [configuration](configuration.md)) can log in and obtain an access token.

You also need to set up the [REST Auth](https://github.com/ma1uta/matrix-synapse-rest-password-provider) password provider module and point it to `matrix-corporal`'s HTTP gateway server's `/_matrix/corporal` endpoint (e.g. `http://matrix-corporal:41080/_matrix/corporal`). This is necessary for making Interactive Authentication (initiated by the homeserver) work when it's `matrix-corporal` that handles [user authentication](user-authentication.md).

You should also make sure that the federation port (8448) of Matrix Synapse only handles federation traffic (not `client` API traffic). By default, it probably handles both (see Synapse's `listeners` section in `homeserver.yaml`), so you need to adjust the configuration.

To prevent the `matrix-corporal` user (and other users that matrix-corporal may impersonate) from being rate-limited by the homeserver, you may also need to adjust the rate limits (mostly `rc_login`). If it were just the `matrix-corporal` user, something like [Synapse #6286](https://github.com/matrix-org/synapse/issues/6286)) could have been used as well.


## Reverse proxy configuration

Usually, your setup would already contain a reverse proxy server (like nginx) listening on port `443` and forwarding traffic over to Matrix Synapse.

You need to modify its configuration, so that it no longer forwards to Matrix Synapse, but rather forwards everything over to `matrix-corporal`.

The nginx vhost configuration might look something like this:

```conf
server {
	listen 443 ssl http2;
	listen [::]:443 ssl http2;

	#
	# Other configuration here..
	#

	# If you've enabled Matrix Corporal's HTTP API, proxy to the HTTP API server
	location /_matrix/corporal {
		proxy_pass http://localhost:41081;
	}

	# If you're using mxisd, proxy that traffic to the mxisd server
	location /_matrix/identity {
		proxy_pass http://localhost:8090;
	}

	# Proxy all other Matrix traffic to Matrix Corporal's HTTP Gateway server
	location /_matrix {
		proxy_pass http://localhost:41080;
		proxy_set_header X-Forwarded-For $remote_addr;
	}
}
```
