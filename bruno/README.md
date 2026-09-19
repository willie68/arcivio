# Bruno collection: arcivio

Open this folder (`bruno/`) in Bruno. URLs are collection defaults in `collection.bru`. Environments only override ports. Select `local` for `configs/service_local.yaml`.

| Environment | Config | HTTP | HTTPS |
| --- | --- | --- | --- |
| `local` | `configs/service_local.yaml` | 9480 | 9443 |
| `testdata` | `testdata/service_local.yaml` | 9000 | 9443 |
| `minimal` | `testdata/service_local_minimal.yaml` | 8000 | 8543 |

With TLS enabled, the HTTP port only serves `/livez`, `/readyz`, `/` and `/metrics`. The SPA and REST API run against `apiUrl` (HTTPS).

The service uses a generated or file-based certificate. Disable TLS verification in Bruno, or run:

```
bru run --insecure --env local
```

JWT is enabled in the default local config (`auth.type: jwt`). Unauthenticated
calls to `/api/v1/*` return 401. Log in via the SPA (OIDC authorization code + PKCE)
or obtain a token from `POST /auth/token`.
