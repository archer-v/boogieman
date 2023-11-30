# boogieman
The probing utility (and golang library) to monitor the availability of host nodes, networks, services and processes.
It's intended as a lightweight simple utility as part of automation scripts in different DevOPS scenarios and NOC working processes. All probes and scenarios expose their data as Prometheus metrics and this utility can be used as source of hosts, networks and services availability metrics.

Two working modes available: console utility for single scenario or probe run or a daemon for regular scheduled probing. 

Utility can perform single or composite checks combined in a scenario described in configuration file in a YAML format

Available checks (probes): 
- ping
- web (GET request with response code checking) 
- openvpnConnect
- cmd (arbitrary console command with exit code checking)
- traceroute (with expecting a host in a route)
- any additional probes can be created

All probes can returns addiotional data, like timings, routes, etc. 

Working modes
- console mode: with text or JSON output
- continuos monitoring mode: perform regular checks and exposes the results as prometheus metrics or json

Probes in a scenario can be configured to execute simultaneously so the entire scenario can perform quickly. Configurable timeouts are supported for all checks. 

Scenario file example:

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
        expect: false
        timeout: 200
      configuration:
        hosts:
          - 192.168.105.105
```

Regular jobs configuration file example:
```
global:
  default_schedule: 60s #execute every 60 seconds
  bind_to: localhost:9091
jobs:
  - script: test/script-openvpn.yml
    name: TestJob1
    schedule: 10 * * * * * #sec, min, hour, day, month, day of week
    timeout: 30000
  - script: test/script-simple.yml
    name: TestJob2
    timeout: 10000
```

HTTP api endpoints:
* /job?name=job_name - returns JSON object with result of a last job execution
* /jobs - returns current job list in schedule queue


/job response example:
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

/jobs responce example:
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
