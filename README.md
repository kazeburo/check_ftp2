# check_ftp2

Nagios check_ftp plugin alternative supporting TLSv1.2 and explicit/implicit TLS mode.
Not implemented full feature, only we need.

## Usage

```
.Usage:
  check_ftp2 [OPTIONS]

Application Options:
      --timeout=  Timeout to wait for connection (default: 10s)
  -H, --hostname= IP address or Host name (default: 127.0.0.1)
  -p, --port=     Port number (default: 21)
  -S, --ssl       use TLS
      --sni=      sepecify hostname for SNI
      --explicit  Use Explicit TLS mode
  -4              use tcp4 only
  -6              use tcp6 only
  -v, --version   Show version

Help Options:
  -h, --help      Show this help message
```

