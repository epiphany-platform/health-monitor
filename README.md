**Healthd**

Daemon implementing health check monitor probes for Epiphany platform.

**Health Checks**

Health checks are a means of regularly monitoring the health of individual components within the Epiphany environment. These monitoring probes checks the liveness of containers, services, processes at specified intervals and, in the event of an unhealthy object take predefined action.

**Features**

healthd is an Epiphany Linux based service process providing health check probes, this process is instantiated at system startup, managed and supervised by the service manager (Systemd), and runs unobtrusively in the background throughout its lifecycle. Initial support will be provided for the following Linux based distros:

- CentOS 9
- Ubuntu 18.04
- Redhat 7.6

**Usage**

The Health Check Daemon will implements Linux D-Bus IPC framework protocol (sd\_notify) as defined [https://www.freedesktop.org/software/systemd/man/sd\_notify.html#Description](https://www.freedesktop.org/software/systemd/man/sd_notify.html#Description)for new daemons, which provides robust mechanism for Daemon management and control using systemctl.

**healthd.service** 

You should copy your healthd.service file to /etc/systemd/system. Do not symlink it. One, you can't run systemctl enable because you it will not follow a symlink. Two, it potentially opens up a security risk. 

ExecStart specified the program and arguments to execute when the service is started. healthd should be copied into direftory /usr/sbin and the directory where the healthd.ymp should be specified.

ExecStart=/usr/sbin/healthd -c /home/smeadows/Documents/golang/src/github.com/healthd/healthd.yml

**healthd.yml**  **Config**

The Health Check Daemon configuration file format will be based upon YAML to provide key-value pairs in human-readable format.

| **Key** | **Description** | **Value** |
| --- | --- | --- |
| Name | Specifies the associated application name | Unique defined string |
| Package | Specifies the Golang package name. | Currently supported HTTP, Docker and Prometheus. |
| Interval | Specifies the probe interval in seconds. | >= 10. Default 10. |
| Retries | Specified the number of times to retry probe after first failure. | >= 3. Default 7. |
| RetryDelay | Specifies delay time in seconds between retry attempt. | Must be greater than 3 seconds. |
| ActionFatal | Specifies whether to KILL associated DAEMON. | True/false default false |
| IP | Specifies IP address of associated probed daemon. | Endpoint IP address. |
| Port | Specifies the associated IP address port number associated with probed daemon. | Endpoint port number |
| Path | Consist of a sequence of path segments separated by a slash (/) | Endpoint path |
| RequestType | Specifies the HTTP method to be used for probing associated daemon. | head, or get, Default head. |
| Response | Specifies the associated good response &quot;200 Ok&quot;. | Optional, default 200. |

**Controlling Service** 
 **Control whether service loads on boot**

sudo systemctl enable healthd.
sudo systemctl disable healthd.

**See if running, uptime, view latest logs**

sudo systemctl status.
sudo systemctl status healthd.
 
 **Show syslog for service**

journalctl --unit healthd.
