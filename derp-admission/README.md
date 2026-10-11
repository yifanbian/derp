# derp-admission

Admission control for [Tailscale DERP servers](https://tailscale.com/kb/1118/custom-derp-servers). A small JSON API server periodically fetches the device list of the configured tailnets, caches their node keys, and answers `derper`'s `-verify-client-url` requests so that only devices from those tailnets are admitted.

## Configuration

See [config-example.yaml](config-example.yaml):

```yaml
listen: ":8080"        # optional, default :8080
refreshInterval: 5m    # optional, default 5m, minimum 1s
tailnets:
- name: tailnet1       # tailnet name
  auth:
    clientCredential:  # OAuth client secret
      id: client1
      secret: tskey-client-xxxx
- name: tailnet2
  auth:
    identityFederation:  # currently only Azure IMDS managed identity is supported
      id: client2        # ID of the trust credential on the Tailscale side
      azureImds:
        clientId: 00000000-0000-0000-0000-000000000000  # optional, client ID of a user-assigned identity
```

Each tailnet must set exactly one of `clientCredential` and `identityFederation`.

### Tailscale credentials

Both methods only need the `devices:core:read` scope; grant nothing more when creating the credential in the Tailscale admin console.

- **clientCredential**: create an OAuth client and put its ID and secret in the config.
- **identityFederation**: create a trust credential with Azure as the identity provider and trust your managed identity. The server requests a token from Azure IMDS with audience `api://AzureADTokenExchange` and exchanges it for a Tailscale API token, so it must run on an Azure resource that has a managed identity.

## Run

```sh
go build -o derp-admission .
./derp-admission -config config.yaml        # add -debug for debug logs
```

Docker (multi-arch image, linux/amd64 and linux/arm64). The image runs as UID 65532, so the mounted config must be readable by it:

```sh
docker run -d -p 8080:8080 \
  -v $PWD/config.yaml:/config.yaml:ro \
  ghcr.io/<owner>/<repo>/derp-admission:latest
```

## Use with derper

```sh
derper -verify-client-url=http://<host>:8080/
```

Optionally pass `-verify-client-url-fail-open=false` to reject all clients when this service is unreachable (by default clients are admitted).

## API

`POST /` with a `tailcfg.DERPAdmitClientRequest` body; the response is `{"Allow": true|false}`. A node is allowed if its key is cached for any configured tailnet.

## Behavior

- Refreshes once at startup, then every `refreshInterval`.
- If a tailnet fails to refresh, its previous data is kept and the error is logged.
- If the first refresh fails the server still starts and denies every node until a refresh succeeds.
- A deleted device stops being admitted at most one refresh interval later.
