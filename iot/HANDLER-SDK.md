# KINGAI OS Edge Handler SDK

Board handlers are the only components that should receive the Linux permissions needed to access hardware. `kingaid` remains non-root and `PrivateDevices=yes`.

Use `sdk/edgehandler` for Go handlers or implement the same protocol described in `HANDLER-PROTOCOL.md`.

## Security contract

A handler must:

- listen only on `/run/kingai-device/<handler>.sock`;
- expose only AF_UNIX, never a management TCP port by default;
- make the socket mode `0600` and chown it to the `kingaid` UID so only the broker can connect;
- declare an exact capability/resource allowlist in its own process;
- reject wildcard, path and shell-like resource syntax;
- avoid shell interpretation of `target` or `arguments`;
- run with only the device nodes, groups and Linux capabilities it actually needs;
- return structured JSON rather than stdout/stderr command output;
- place physical actuators in a safe state on crash, timeout or shutdown.

## Minimal Go shape

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

err := edgehandler.Serve(ctx, edgehandler.Config{
    SocketPath:  "/run/kingai-device/gpio-read.sock",
    SocketOwner: kingaiUID,
    Capabilities: map[string][]string{
        "device.gpio.read": {"gpio:17", "gpio:18"},
    },
}, edgehandler.HandlerFunc(func(ctx context.Context, req edgehandler.Request) (edgehandler.Result, error) {
    // Call a typed GPIO library here. Never execute req.Target as a command.
    return edgehandler.Result{Data: json.RawMessage(`{"value":1}`)}, nil
}))
```

The Device Pack must independently declare the same capability/resources. This duplication is intentional defense in depth.

## Suggested handler split

Prefer small services instead of one omnipotent hardware daemon, for example:

- `telemetry-read`
- `gpio-read` / `gpio-write`
- `i2c-read` / `i2c-write`
- `spi-read` / `spi-write`
- `camera-capture`
- `gpu-compute` / `npu-compute`
- `motor-control`
- `power-control`

A board may combine functions only when the operating-system permissions and failure domain are genuinely the same.

## Physical safety

AI policy is not a substitute for a hardware safety circuit. Robots, relays, motors, heaters, chargers and other hazardous actuators need independent limits, watchdogs/interlocks and emergency-stop behavior appropriate to the product. KINGAI's L4/L5 Approval gates reduce software authority but do not replace hardware safety engineering.
