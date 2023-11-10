# boogieman
The probing utility to monitor the availability of host nodes, networks, services and processes.
It's intended to use as a console utility for fast check or as part of automation scripts in different DevOPS scenarios and NOC working processes.  

Utility can perform as single checks (like ping or remote service health check) or composite checks combined in a scenario described in configuration file in a YAML format

If boogieman is started in console mode, it returns the check result in exit code and stdout message. 
In daemon mode it exposes the results as prometheus metrics or json object
