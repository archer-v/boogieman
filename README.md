# boogieman
The probing utility, is designed to monitor the availability of host nodes, networks, services, and processes. It serves as a lightweight and straightforward tool, ideal for integration into automation scripts or as a metrics provider in various DevOps and network operations. All probes within this utility expose their data as Prometheus metrics or a JSON formatted object exposed at a http port, making it a reliable source for availability metrics related to hosts, networks, routes, and network services.

The utility can perform single or composite checks combined in a scenario described in configuration file in a YAML format. Checks can be executed in parallel, so an entire scenario can execute quickly.

### Available checks (probes): 
- ping
- web (GET request with response code checking) 
- openvpn
- cmd (arbitrary console command with exit code checking)
- traceroute (with checking of presense of a particular host in a traceroute)
- any additional probes can be created

All probes can returns additional data, like timings, response codes, stdout, etc

### Working modes
- console mode: performs single scenario or probe run with text or JSON formatted output to stdout
- continuos (daemon) monitoring mode: performs regular checks and exposes the results as prometheus metrics or a json formatted string
```
./boogieman
boogieman - version: devel-main-b61f6e4, build: 2023-12-12_210008

  Usage:
    boogieman [oneRun|daemon]

  Subcommands: 
    oneRun   performs a single run, print result and exit
    daemon   start in daemon mode and performs scheduled jobs

  Flags: 
       --version   Displays the program version string.
    -h --help      Displays help with available flag, subcommand, and positional value parameters.

```
#### Console mode

Use **oneRun** command option to execute the particular probe or script. The result is output to a console as a plain text or json formatted string. Also a utility is finished with exit code 0 on success checks execution.

For example:
 * `./boogieman oneRun --script test/script-simple.yml -J` will execute the test/script-simple.yml script and output the result to the console in JSON format
 * `./boogieman oneRun --probe ping -c msn.com,github.com` will perform ping of two hosts msn.com,github.com and output the result to the console.

Run options: 
```
./boogieman oneRun
oneRun - performs a single run, print result and exit

  Flags: 
       --version   Displays the program version string.
    -h --help      Displays help with available flag, subcommand, and positional value parameters.
    -s --script    path to a script file in yml format
    -p --probe     single probe to start (ignored if script option is selected)
    -c --config    probe configuration string (ignored if script option is selected)
    -t --timeout   probe waiting timeout (ignored if script option is selected) (default: 0s)
    -d --debug     debug logging
    -v --verbose   verbose logging
    -e --expect    expected result true|false (ignored if script option is selected) (default: true)
    -j --json      output result in JSON format
    -J --jsonp     output result in JSON format with indents and CR
```

#### Daemon mode

`./boogieman daemon --config test/boogieman.yml` will start the utility in a daemon mode with configuration read from test/boogieman.yml. 

Configuration file contains global options, list of jobs (scripts) and a schedule of their execution. HTTP server is listen at the tcp port (default 9091) and exposes scripts execution result as a prometheus metrics and json format text.

Available http endpoints:

* /job?name=job_name - returns JSON object with result of a last job execution
* /jobs - returns a job list in a schedule queue
* /metrics - prometheus metrics

### Concurrent task execution and timeouts

Several probes in a script can be configured to execute simultaneously so an entire scenario can execute quickly. Configurable timeouts are supported for all checks. Use cgroup option with the same value to define concurrent group.

### Configuration examples

**Script file:**
```
script:
  - name: gateway-alive
    cgroup: 1
    probe:
      name: ping
      options:
        timeout: 100
      configuration:
        # localhost just for an example
        hosts:
          - 127.0.0.1
          - 127.0.0.2
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
        # 192.168.105.105 host shouldn't ping
        expect: false
        timeout: 200
      configuration:
        hosts:
          - 192.168.105.105
```

**Daemon configuration file:**
```
global:
  default_schedule: 60s
  bind_to: localhost:9091
jobs:
  - script: test/script-openvpn.yml
    name: TestJob1
    schedule: 10 * * * * * #sec, min, hour, day, month, day of week
    timeout: 30000
    vars:
      vpn-connect:
        remote_host: 127.0.0.3
  - script: test/script-simple.yml
    name: TestJob2
    timeout: 10000
    vars:
      gateway-alive:
        hosts: 127.0.0.3, 127.0.0.4
      internet-alive:
        urls: https://msn.com/
```

### Example of responses 

http /job endpoint response example:
```
{
   "result":{
      "startedAt":"2023-12-01T00:37:08.208525151+05:00",
      "runtime":799,
      "success":true,
      "runCounter":174
   },
   "status":"finished",
   "tasks":[
      {
         "name":"gateway-alive",
         "status":"finished",
         "probe":{
            "name":"ping",
            "options":{
               "timeout":100,
               "expect":true
            },
            "startedAt":"2023-12-01T00:37:08.208614247+05:00",
            "runtime":0,
            "success":true,
            "runCounter":174,
            "data":{
               "127.0.0.1":2,
               "127.0.0.2":4
            }
         },
         "startedAt":"2023-12-01T00:37:08.208610199+05:00",
         "runtime":4,
         "success":true,
         "runCounter":174
      },
      {
         "name":"internet-alive",
         "status":"finished",
         "probe":{
            "name":"web",
            "options":{
               "timeout":1500,
               "expect":true
            },
            "startedAt":"2023-12-01T00:37:08.208537003+05:00",
            "runtime":558,
            "success":true,
            "runCounter":174,
            "data":{
               "https://github.com/":84,
               "https://google.com/":558
            }
         },
         "startedAt":"2023-12-01T00:37:08.20853557+05:00",
         "runtime":558,
         "success":true,
         "runCounter":174
      },
      {
         "name":"backup-gateway-disabled",
         "status":"finished",
         "probe":{
            "name":"ping",
            "options":{
               "timeout":200,
               "expect":false
            },
            "startedAt":"2023-12-01T00:37:08.767422984+05:00",
            "runtime":240,
            "success":true,
            "runCounter":174,
            "data":{
               
            }
         },
         "startedAt":"2023-12-01T00:37:08.767421631+05:00",
         "runtime":240,
         "success":true,
         "runCounter":174
      }
   ]
}
```

http /jobs endpoint response example:
```
[
   {
      "name":"TestJob1",
      "script":"test/script-openvpn.yml",
      "schedule":"10 * * * * *",
      "once":false,
      "timeout":30000,
      "nextStartAt":"2023-11-30T22:33:10+05:00"
   },
   {
      "name":"TestJob2",
      "script":"test/script-simple.yml",
      "schedule":"60s",
      "once":false,
      "timeout":10000,
      "nextStartAt":"2023-11-30T22:33:08.180271936+05:00"
   }
]
```

http /metrics response example (prometheus metrics)
```
# HELP boogieman_probe_data_item probe execution data result
# TYPE boogieman_probe_data_item gauge
boogieman_probe_data_item{item="127.0.0.3",job="TestJob2",probe="ping",script="test/script-simple.yml",task="gateway-alive"} 0
boogieman_probe_data_item{item="127.0.0.4",job="TestJob2",probe="ping",script="test/script-simple.yml",task="gateway-alive"} 2
boogieman_probe_data_item{item="https://msn.com/",job="TestJob2",probe="web",script="test/script-simple.yml",task="internet-alive"} 1237
# HELP boogieman_script_result script execution result
# TYPE boogieman_script_result gauge
boogieman_script_result{job="TestJob1",script="test/script-openvpn.yml"} 0
boogieman_script_result{job="TestJob2",script="test/script-simple.yml"} 1
# HELP boogieman_task_result task execution result
# TYPE boogieman_task_result gauge
boogieman_task_result{job="TestJob1",script="test/script-openvpn.yml",task="gateway-alive"} 0
boogieman_task_result{job="TestJob1",script="test/script-openvpn.yml",task="tunnel-network-routing"} 0
boogieman_task_result{job="TestJob1",script="test/script-openvpn.yml",task="vpn-connect"} 0
boogieman_task_result{job="TestJob1",script="test/script-openvpn.yml",task="vpn-tunnel-alive"} 0
boogieman_task_result{job="TestJob2",script="test/script-simple.yml",task="backup-gateway-disabled"} 1
boogieman_task_result{job="TestJob2",script="test/script-simple.yml",task="gateway-alive"} 1
boogieman_task_result{job="TestJob2",script="test/script-simple.yml",task="internet-alive"} 1
# HELP boogieman_task_runs task run counter
# TYPE boogieman_task_runs counter
boogieman_task_runs{job="TestJob1",script="test/script-openvpn.yml",task="gateway-alive"} 0
boogieman_task_runs{job="TestJob1",script="test/script-openvpn.yml",task="tunnel-network-routing"} 0
boogieman_task_runs{job="TestJob1",script="test/script-openvpn.yml",task="vpn-connect"} 0
boogieman_task_runs{job="TestJob1",script="test/script-openvpn.yml",task="vpn-tunnel-alive"} 0
boogieman_task_runs{job="TestJob2",script="test/script-simple.yml",task="backup-gateway-disabled"} 1
boogieman_task_runs{job="TestJob2",script="test/script-simple.yml",task="gateway-alive"} 1
boogieman_task_runs{job="TestJob2",script="test/script-simple.yml",task="internet-alive"} 1
# HELP boogieman_task_runtime task runtime
# TYPE boogieman_task_runtime gauge
boogieman_task_runtime{job="TestJob1",script="test/script-openvpn.yml",task="gateway-alive"} 0
boogieman_task_runtime{job="TestJob1",script="test/script-openvpn.yml",task="tunnel-network-routing"} 0
boogieman_task_runtime{job="TestJob1",script="test/script-openvpn.yml",task="vpn-connect"} 0
boogieman_task_runtime{job="TestJob1",script="test/script-openvpn.yml",task="vpn-tunnel-alive"} 0
boogieman_task_runtime{job="TestJob2",script="test/script-simple.yml",task="backup-gateway-disabled"} 253
boogieman_task_runtime{job="TestJob2",script="test/script-simple.yml",task="gateway-alive"} 2
boogieman_task_runtime{job="TestJob2",script="test/script-simple.yml",task="internet-alive"} 1237
boogieman_task_runtime{job="TestJob3",script="test/script-cmd.yml",task="gateway-alive-cmd-ping"} 313
```

### Note

RAW_SOCKETS are used to perform ping and traceroute checks. So, the command requires root privileges to perform this checks. You can user sudo, or you can grant permissions only for operation with a raw sockets by setting the SET_CAP_RAW flag on the executable file. Use setcap command: `setcap cap_net_raw+ep ./boogieman`
