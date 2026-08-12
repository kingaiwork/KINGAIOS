# Controlled Release Requests

KINGAI OS does not create a release on every code push.

A Server Developer Preview can be explicitly requested by updating `.release/server-dev.json`. That request triggers the same-runner **build → ownership validation → ISO generation → checksum/manifest validation → QEMU boot test → size routing → Pre-release publication** workflow.

RC and Stable channels remain blocked by release gates until install-to-disk, production Secure Boot signing, atomic A/B update activation and rollback validation are complete.

Large artifacts (>=2 GiB) require the R2 repository secrets and are routed to the KINGAI object store. Secrets are never committed to this directory or repository.
