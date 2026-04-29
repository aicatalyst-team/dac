# Installation

DAC is released as standalone binaries through GitHub Releases.

## Install Script

```shell
curl -fsSL https://raw.githubusercontent.com/bruin-data/dac/main/install.sh | bash
```

The installer installs the Bruin CLI first if `bruin` is not already available on your `PATH`, then downloads the latest DAC GitHub release for your platform and installs `dac` into `~/.local/bin` by default.

DAC uses `.bruin.yml` connections and currently shells out to `bruin query` to execute dashboard SQL.

To install the latest edge build from `main`:

```shell
curl -fsSL https://raw.githubusercontent.com/bruin-data/dac/main/install.sh | bash -s -- --channel edge
```

To install into a different directory:

```shell
curl -fsSL https://raw.githubusercontent.com/bruin-data/dac/main/install.sh | bash -s -- -b /usr/local/bin
```

To install a specific version:

```shell
curl -fsSL https://raw.githubusercontent.com/bruin-data/dac/main/install.sh | bash -s -- v0.1.0
```

## Build from Source

If you are developing DAC itself:

```shell
git clone https://github.com/bruin-data/dac.git
cd dac
make deps
make build
```

The binary is output under the repository `bin/` directory. Add that directory to your `PATH` if you want to run it as `dac`.

## Verify Installation

```shell
dac version
```

If you will run dashboards locally, also verify that the Bruin CLI is on your `PATH`:

```shell
bruin --version
```
