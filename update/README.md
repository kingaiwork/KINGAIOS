# KINGAI OS Update Trust

Official KINGAI OS updates are designed to be authenticated before installation and then staged into a rollback-capable system slot.

The Developer Foundation includes an Ed25519 manifest verifier and artifact SHA-256/size verification. Production root signing keys are intentionally **not** stored in this repository. Root/release key generation, offline custody, threshold policy, TUF metadata and A/B activation remain gated release-engineering work.

No update component in the Developer Foundation performs destructive partition switching automatically.
