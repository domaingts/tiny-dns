# tiny-dns

`tiny-dns` is a small UDP DNS proxy that asks an upstream resolver for `AAAA` records and falls back to a fixed IPv4 address when no IPv6 answers are returned.

It is useful when you want names with real IPv6 records to resolve normally, while everything else is mapped to a predictable IPv4 address inside your network.

## What It Does

For each incoming DNS request, the server:

1. Reads the first question from the message.
2. Looks for a cached synthesized fallback response for that name.
3. Sends an upstream `AAAA` lookup over UDP.
4. Copies any `AAAA` answers from the upstream response into the reply.
5. If no `AAAA` answers are returned, synthesizes a single `A` record using the configured default IP.

Current implementation details:

- The server listens on UDP only.
- The upstream lookup is always `AAAA`, regardless of the client query type.
- If no IPv6 answers are found, the fallback answer is always an `A` record with TTL `300`.
- Only synthesized fallback responses are cached in-process.
- Only the first question in a DNS message is handled.

## Requirements

- Go `1.26.2` to build from source

## Build

```bash
go build .
```

This produces a `tiny-dns` binary in the repository directory.

## Run

```bash
go run . \
  -addr :15553 \
  -dns 1.1.1.1:53 \
  -default 10.1.11.111
```

Or run the compiled binary:

```bash
./tiny-dns \
  -addr :15553 \
  -dns 1.1.1.1:53 \
  -default 10.1.11.111
```

## Flags

| Flag | Default | Description |
| --- | --- | --- |
| `-addr` | `:15553` | UDP listen address for the DNS server |
| `-dns` | `1.1.1.1:53` | Upstream DNS resolver address |
| `-default` | `10.1.11.111` | IPv4 address returned when no upstream `AAAA` answer exists |

## Example Queries

Start the server:

```bash
go run . -addr :15553 -dns 1.1.1.1:53 -default 10.1.11.111
```

Query it with `dig`:

```bash
dig @127.0.0.1 -p 15553 example.com AAAA
```

Behavior:

- If the upstream resolver returns `AAAA` answers, those IPv6 records are sent back.
- If the upstream resolver returns no `AAAA` answers, `tiny-dns` replies with a synthetic `A` record pointing to the configured default IP.

## Notes And Limitations

- The server uses a 1 second timeout for upstream DNS requests.
- Invalid `-default` IP values cause startup to panic.
- Requests are forwarded to the upstream resolver over UDP only.
- The reply keeps the upstream response code, even when a synthetic fallback `A` record is added.
- Release packaging in this repository is configured for Linux `amd64`.

## Development

Build:

```bash
go build .
```

Run tests:

```bash
go test ./...
```

If `go test` fails in your environment with:

```text
testing/internal/testdeps: package testmain: cannot find package
```

that is an issue with the local Go test harness/environment, not with the README changes.
