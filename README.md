# boogieman

Boogieman is a lightweight Go utility for probing the availability of hosts, networks, routes, HTTP services, OpenVPN tunnels, and local processes. It can run a single check, execute a YAML scenario, or work as a daemon that exposes the latest results as JSON and Prometheus metrics.

## Features

- Single-probe and multi-step scenario execution.
- Parallel execution of tasks that belong to the same `cgroup`.
- Console output or JSON output for one-shot runs.
- Daemon mode with scheduled jobs.
- HTTP API for the latest job results.
- Prometheus metrics for script, task, runtime, run counter, and probe data values.
- Extensible probe registry for adding custom probes.

## Available probes

- `ping` - checks host reachability and returns response timings.
- `web` - sends HTTP GET requests and checks the expected status code.
- `cmd` - starts a local command and checks its exit code.
- `openvpn` - starts an OpenVPN client and waits for successful initialization.
- `traceroute` - runs traceroute and checks whether expected hops are present or absent.

## Build

Requirements:

- Go 1.21 or newer.
- Linux target for the default `make build`.
- `openvpn` for OpenVPN checks and related tests.
- Root privileges or `cap_net_raw` for `ping` and `traceroute` checks.

Common commands:

```bash
make dep
make build
make test
make distro
```

`make build` creates `build/boogieman`. The default Makefile build is a static Linux build; set `GOARCH` if you need a specific architecture.

## Usage

```text
boogieman [oneRun|daemon]

Subcommands:
  oneRun   performs a single run, prints the result, and exits
  daemon   starts daemon mode and performs scheduled jobs
```

### One-shot mode

Use `oneRun` to execute either a YAML script or one probe passed through the CLI.

Examples:

```bash
./boogieman oneRun --script test/script-simple.yml -J
./boogieman oneRun --probe ping --config msn.com,github.com
./boogieman oneRun --probe web --config https://example.com --timeout 2s --json
```

Options:

```text
-s, --script    path to a script file in YAML format
-p, --probe     single probe to start; ignored when --script is used
-c, --config    probe configuration string; ignored when --script is used
-t, --timeout   probe timeout; ignored when --script is used
-d, --debug     debug logging
-v, --verbose   verbose logging
-e, --expect    expected result flag; ignored when --script is used
-j, --json      compact JSON output
-J, --jsonp     pretty JSON output
```

Exit codes:

- `0` - check succeeded.
- `1` - check ran but returned an unsuccessful result.
- `2` - startup or configuration error.

### Daemon mode

Daemon mode reads a YAML configuration with global options and scheduled jobs.

```bash
./boogieman daemon --config test/boogieman.yml
```

The HTTP server listens on `global.bind_to`; the default is `localhost:9091`.

Endpoints:

- `/job?name=<job_name>` - latest finished result for a job.
- `/jobs` - configured jobs and their next start time.
- `/metrics` - Prometheus metrics.

Schedules can be either Go duration strings such as `60s` or cron expressions with seconds such as `10 * * * * *`.

## Scenario execution

A script is a sequence of tasks. Each task wraps one probe.

Tasks with the same `cgroup` value are executed in parallel. Groups are executed sequentially in the order they appear in the script. If `cgroup` is omitted, Boogieman assigns an internal group automatically.

Probe option `timeout` in YAML is expressed in milliseconds.

## Configuration examples

### Script file

```yaml
script:
  - name: gateway-alive
    cgroup: 1
    probe:
      name: ping
      options:
        timeout: 100
      configuration:
        hosts:
          - 127.0.0.1
          - 127.0.0.2
    metric:
      labels:
        environment: test
      valueMap:
        127.0.0.1: host1
        127.0.0.2: host2

  - name: internet-alive
    cgroup: 1
    probe:
      name: web
      options:
        timeout: 1500
      configuration:
        urls:
          - https://google.com/
          - https://github.com/
        httpStatus: 200

  - name: backup-gateway-disabled
    probe:
      name: ping
      options:
        expect: false
        timeout: 200
      configuration:
        hosts:
          - 192.168.105.105
```

### Daemon configuration

```yaml
global:
  default_schedule: 60s
  bind_to: localhost:9091
  exit_on_config_change: true

jobs:
  - script: test/script-openvpn.yml
    name: TestJob1
    schedule: 10 * * * * *
    timeout: 30000
    vars:
      vpn-connect:
        configFile: src/probes/openvpn/test/openvpn-client.ovpn

  - script: test/script-simple.yml
    name: TestJob2
    timeout: 10000
    vars:
      gateway-alive:
        hosts: 127.0.0.3, 127.0.0.4
      internet-alive:
        urls: https://msn.com/
```

`vars` can override probe configuration fields by task name. Values are parsed as strings and converted to the target field type where supported.

## Probe configuration reference

### ping

```yaml
probe:
  name: ping
  options:
    timeout: 1000
    expect: true
  configuration:
    hosts:
      - 127.0.0.1
    interval: 500
```

`interval` is in milliseconds. The probe returns a map of host names to response times in milliseconds.

### web

```yaml
probe:
  name: web
  options:
    timeout: 1500
  configuration:
    urls:
      - https://example.com/
    httpStatus: 200
    fwMark: 100
    bodyRegex: "service version: [0-9]+\\.[0-9]+"
    bodyRegexInvert: false
    bodyRegexCaptureGroup: 1
```

URLs without a scheme are treated as HTTPS URLs.

`httpStatus` is optional. If it is omitted or set to `0`, the HTTP status code is not checked and does not affect probe success. Response timings and returned HTTP status codes are exported whenever the endpoint returns an HTTP response, even when another probe condition fails.

On Linux, `fwMark` sets `SO_MARK` on sockets opened by the web probe. This can be used with `ip rule` and policy routing. On non-Linux systems, `fwMark` is ignored and the probe uses the regular HTTP client behavior.

If `bodyRegex` is set, the probe reads the response body and checks it against the regular expression after the endpoint returns an HTTP response. With `bodyRegexInvert: false`, the probe succeeds only when the expression matches. With `bodyRegexInvert: true`, the probe succeeds only when the expression does not match.

If `bodyRegexCaptureGroup` is greater than `0`, the selected capture group is returned in the probe result data together with response timings. Capture groups cannot be used together with `bodyRegexInvert`.

### cmd

```yaml
probe:
  name: cmd
  options:
    timeout: 500
    stayBackground: false
  configuration:
    cmd: ping -c 3 -i 0.1 -W 0.1 127.0.0.1
    exitCode: 0
    logDump: false
    stdoutRegex: "bytes from"
    stdoutRegexInvert: false
    stdoutRegexCaptureGroup: 1
```

`stayBackground: true` means the command is expected to keep running after the startup timeout. If it exits earlier, the probe fails.

If `stdoutRegex` is set, the probe checks the command stdout after the command exits and the exit code matches. With `stdoutRegexInvert: false`, the probe succeeds only when stdout matches the expression. With `stdoutRegexInvert: true`, the probe succeeds only when stdout does not match.

If `stdoutRegexCaptureGroup` is greater than `0`, the selected capture group is returned in the probe result data together with the exit code. Capture groups cannot be used together with `stdoutRegexInvert`.

### openvpn

```yaml
probe:
  name: openvpn
  options:
    timeout: 5000
    stayBackground: true
  configuration:
    configFile: src/probes/openvpn/test/openvpn-client.ovpn
    logDump: false
```

Use `configData` instead of `configFile` to pass OpenVPN configuration content directly.

### traceroute

```yaml
probe:
  name: traceroute
  options:
    timeout: 2000
  configuration:
    host: 8.8.8.8
    expectedHops:
      - 8.8.8.8
    expectedMatch: any
    maxHops: 30
    retries: 2
```

`expectedMatch` supports `any`, `all`, and `none`.

## Response examples

### `/job`

```json
{
  "result": {
    "startedAt": "2023-12-01T00:37:08.208525151+05:00",
    "runtime": 799,
    "success": true,
    "runCounter": 174
  },
  "status": "finished",
  "tasks": [
    {
      "name": "gateway-alive",
      "status": "finished",
      "probe": {
        "name": "ping",
        "options": {
          "timeout": 100,
          "expect": true
        },
        "runtime": 4,
        "success": true,
        "runCounter": 174,
        "data": {
          "127.0.0.1": 2,
          "127.0.0.2": 4
        }
      },
      "runtime": 4,
      "success": true,
      "runCounter": 174
    }
  ]
}
```

### `/jobs`

```json
[
  {
    "name": "TestJob2",
    "script": "test/script-simple.yml",
    "schedule": "60s",
    "once": false,
    "timeout": 10000,
    "nextStartAt": "2023-11-30T22:33:08.180271936+05:00"
  }
]
```

### `/metrics`

```text
# HELP boogieman_probe_data_item probe execution data result
# TYPE boogieman_probe_data_item gauge
boogieman_probe_data_item{item="127.0.0.3",job="TestJob2",probe="ping",script="test/script-simple.yml",task="gateway-alive"} 0
boogieman_probe_data_item{item="https://msn.com/",job="TestJob2",probe="web",script="test/script-simple.yml",task="internet-alive"} 1237

# HELP boogieman_script_result script execution result
# TYPE boogieman_script_result gauge
boogieman_script_result{job="TestJob2",script="test/script-simple.yml"} 1

# HELP boogieman_task_result task execution result
# TYPE boogieman_task_result gauge
boogieman_task_result{job="TestJob2",script="test/script-simple.yml",task="gateway-alive"} 1

# HELP boogieman_task_runtime task runtime
# TYPE boogieman_task_runtime gauge
boogieman_task_runtime{job="TestJob2",script="test/script-simple.yml",task="internet-alive"} 1237

# HELP boogieman_task_runs task run counter
# TYPE boogieman_task_runs counter
boogieman_task_runs{job="TestJob2",script="test/script-simple.yml",task="internet-alive"} 1
```

## Adding a probe

To add a probe:

1. Create a package under `src/probes/<name>`.
2. Implement `model.Prober`, usually by embedding `model.ProbeHandler`.
3. Implement a `probefactory.Constructor`.
4. Register the constructor with `probefactory.RegisterProbe`.
5. Add a blank import in `src/probes/probes.go`.

## Notes

`ping` and `traceroute` use raw sockets. Run Boogieman as root or grant the binary the required capability:

```bash
sudo setcap cap_net_raw+ep ./boogieman
```

Some tests depend on external network access, raw sockets, and OpenVPN. CI runs them with elevated privileges.
