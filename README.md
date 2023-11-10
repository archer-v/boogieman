# boogieman
The probing utility to monitor the availability of host nodes, networks, services and processes.
It's intended to use as a console utility for fast check or as part of automation scripts in different DevOPS scenarios and NOC working processes.  

Utility can perform single checks or composite checks combined in a scenario described in configuration file in a YAML format

Available checks (probes): 
- ping
- web 
- openvpnConnect
- cmd (in development)
- netroute (in development)
- netport (in development)
Additional probes can be created.

If boogieman is started in console mode, it returns the check result in exit code and stdout message. 
In daemon mode it exposes the results as prometheus metrics or json object

Scenario file example:

```
script:
  - name: gateway-alive
    probe:
      name: ping
      options:
        timeout: 100
      configuration:
        # localhost just for a test example
        hosts:
          - 127.0.0.1
          - 127.0.0.2
  - name: web-service-alive
    probe:
      name: web
      options:
        timeout: 1500
      configuration:
        urls:
          - https://google.com/
        httpStatus: 200
  - name: backup-gateway-off
    probe:
      name: ping
      options:
        expect: false
        timeout: 200
      configuration:
        hosts:
          - 192.168.105.105
  - name: vpn-connect
    probe:
      name: openvpnConnect
      options:
        timeout: 5000
      configuration:
        configFile: src/probes/openvpnConnect/test/openvpn-client.ovpn
```
