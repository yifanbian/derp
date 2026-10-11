# derp

A DERP server (`derper`) that only admits devices from your own tailnets, plus the pieces to run it.

| Path | Contents |
| --- | --- |
| [derp-admission/](derp-admission) | Admission control server (Go); see its [README](derp-admission/README.md) |
| [derper/](derper) | Dockerfile that builds `tailscale.com/cmd/derper` at a given version |
| [.github/workflows/](.github/workflows) | CI for the admission server; scheduled builds of upstream `derper` |

## Deploy with Docker Compose

`derper` asks `admission` whether each client may connect. The two containers share the Compose network, and `admission` is not published to the host.

`compose.yaml`:

```yaml
services:
  admission:
    image: ghcr.io/<owner>/<repo>/derp-admission:latest
    restart: unless-stopped
    volumes:
      - ./config.yaml:/config.yaml:ro

  derper:
    image: ghcr.io/<owner>/<repo>/derper:latest
    restart: unless-stopped
    depends_on:
      - admission
    command:
      - -hostname=derp.example.com
      - -verify-client-url=http://admission:8080/
      - -verify-client-url-fail-open=false
    ports:
      - "80:80"
      - "443:443"
      - "3478:3478/udp"
    volumes:
      - derper-certs:/var/lib/derper

volumes:
  derper-certs:
```

`config.yaml` is the admission config (see [derp-admission/config-example.yaml](derp-admission/config-example.yaml)); it must be readable by UID 65532 (`chmod 644 config.yaml`). Then:

```sh
docker compose up -d
```

Replace `<owner>/<repo>` with the GitHub repository publishing the images and `derp.example.com` with a hostname that resolves to this machine; `derper` obtains its certificate from Let's Encrypt on ports 80/443. With `-verify-client-url-fail-open=false`, clients are rejected whenever `admission` is unreachable.

## derper image

[derper/Dockerfile](derper/Dockerfile) builds `tailscale.com/cmd/derper` for amd64 and arm64. The version is a required build arg:

```sh
docker build --build-arg DERPER_VERSION=v1.80.2 -t derper derper/
```

`DERPER_VERSION` accepts anything `go install tailscale.com/cmd/derper@<version>` accepts.

The [derper workflow](.github/workflows/derper.yaml) runs daily, looks up the latest `tailscale/tailscale` release, and if `ghcr.io/<owner>/<repo>/derper:<version>` (without the leading `v`) does not exist yet, builds and pushes it, also tagging it `latest`. It can be run manually with a specific `version` input, in which case `latest` is not updated.
