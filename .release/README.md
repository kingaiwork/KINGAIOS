# Controlled Release Requests

KINGAI OS does not create a release on every code push.

Explicit controlled requests:

- `.release/server-dev.json` — Server Developer Preview
- `.release/server-beta.json` — Server Beta
- `.release/desktop-dev.json` — installable Desktop / personal-PC Developer Preview
- `.release/desktop-beta.json` — Desktop Beta after the functional release gates are satisfied

A request triggers the same-runner **build → ownership validation → ISO generation → checksum/manifest validation → QEMU boot test → size routing → Pre-release publication** workflow.

Desktop images are built as installable Live ISOs. The installer provisions GPT + EFI + A/B root partitions + encrypted KINGAI state. Desktop install-to-disk and graphical boot are additionally exercised by `smoke-installable-desktop-iso.yml`.

RC and Stable channels remain fail-closed until production Secure Boot signing, production TUF/release-key custody, protected-branch governance, delivery and real-hardware validation are complete.

Large artifacts (>=2 GiB) require the R2 repository secrets and are routed to the KINGAI object store. Secrets are never committed to this directory or repository.
