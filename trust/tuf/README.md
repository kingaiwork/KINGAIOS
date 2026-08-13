# KINGAI OS TUF Trust

KINGAI OS uses The Update Framework (TUF) for remote update metadata. The runtime client is based on the official `github.com/theupdateframework/go-tuf/v2` library.

## Bootstrap rule

The installed client **must never use Trust On First Use for production updates**. `root.json` is provisioned out-of-band with the OS image or during a separately authenticated root rotation. A network repository is never allowed to choose the initial trust root.

Expected installed location:

```text
/usr/share/kingai/trust/tuf/root.json
```

Mutable verified metadata and downloaded targets live under encrypted STATE:

```text
/var/lib/kingai-state/tuf/metadata
/var/lib/kingai-state/tuf/targets
```

## Roles

Production keys are separated by TUF role:

- **root** — offline, highest-trust, threshold signing; not stored in GitHub Actions or Cloudflare.
- **targets** — release target authorization; preferably hardware-backed/offline approval for RC/Stable.
- **snapshot** — online release metadata consistency role.
- **timestamp** — short-lived online freshness role.

Root rotation follows the TUF specification's sequential root-version verification. All historical root metadata required by clients must remain publishable.

## KINGAI production policy

- Root threshold target: **2-of-3** independent offline keys.
- Root key material: **never committed, never stored in a public CI secret, never uploaded to R2**.
- Targets: separate from timestamp/snapshot credentials.
- Timestamp expiration: short-lived; CI must fail if expiration generation is unavailable.
- RC/Stable publishing is blocked unless `root.json`, all four top-level roles, target metadata, hashes and expiration checks have passed.
- HTTPS is required for availability/privacy, but artifact authenticity derives from TUF metadata rather than TLS alone.

## Developer CI

CI may create ephemeral TUF keys and repositories solely to exercise the updater. Such roots must contain a visible `developer-ci` marker and must never be accepted by RC/Stable publication gates.

## Production secrets that remain external

The repository intentionally contains **no production private key**. Final deployment requires operator-provisioned signing material/HSM identities and the public trusted root. This is an operational security boundary, not unfinished source code.
