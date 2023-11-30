# Bufisk

Download and run `buf` based on configuration from your environment.

Inspired by [Bazelisk](https://github.com/bazelbuild/bazelisk).

Bufisk will use (in order);

- A version specified in the environment variable `${BUF_VERSION}`.
- A version specified in a file named `.bufversion` in your current directory,
  or recursively in any parent directory.

The specified version must be a valid Buf release version from
[github.com/bufbuild/buf/releases](https://github.com/bufbuild/buf/releases).

Bufisk will also download the released `sha256.txt` and `sha256.txt.minisig`,
and do all verification.

All arguments passed to `bufisk` are transparently passed through to `buf`.

```
export BUF_VERSION=1.28.1
bufisk lint
echo 1.28.1 > .bufversion
bufisk lint
```

Bufisk downloads releases to a cache directory. In most Unix-like cases, this will be `~/.cache/bufisk`.

The full logic:

The cache directory specified by `${BUF_CACHE_DIR}`. If `${BUF_CACHE_DIR}` is not set
(as it usually is not), the cache directory defaults to:

- Linux, Darwin: `${XDG_CACHE_HOME}/bufisk`, and if `${XDG_CACHE_HOME}` is not set, `${HOME}/.cache/bufisk`.
- Windows: `%LocalAppData%\bufisk`.

## Status: Alpha

Not yet stable.

Needs some hardening, better error messages, graceful handling
of common error cases, a release process, etc.

We may also want to add support for non-release tags, passing through to
`go install github.com/bufbuild/buf/cmd/buf@${BUF_VERSION}`, and potentially having
special support for a `latest`-type tag for releases. In the `go install case`,
we may also want to bootstrap `go` itself.

This has not yet been tested on Windows.

## Legal

Offered under the [Apache 2 license][license].

[license]: https://github.com/bufbuild/bufisk/blob/main/LICENSE
