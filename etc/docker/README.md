# Matrix Corporal: reconciliator and gateway for a managed Matrix server

[matrix-corporal](https://github.com/devture/matrix-corporal) manages your [Matrix](http://matrix.org/) server according to a configuration policy.

The point is to have a single source of truth about users/rooms/communities somewhere
(say in an external system, like your intranet),
and have something (`matrix-corporal`) continually reconfigure your Matrix server in accordance with it.

# Using this Docker image

Start off by [creating your configuration](https://github.com/devture/matrix-corporal/blob/master/docs/configuration.md).

Since you're running this in a container, make sure `ListenInterface` configuration uses the `0.0.0.0` interface.

To start the container:

```bash
docker run \
-it \
--rm \
-p 127.0.0.1:41080:41080 \
-p 127.0.0.1:41081:41081 \
-v /local/path/to/config.json:/config.json:ro \
devture/matrix-corporal:latest
```

With the above call, ports `41080` and `41081` are only available locally, as you'd most likely run `matrix-corporal` behind a reverse proxy.

**Hint**: using a tagged/versioned image, instead of `latest` is recommended.