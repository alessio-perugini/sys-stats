# sys-status

A GoLang script that send snmp GET request to retrieve CPU load and free available RAM

## Binaries

You can find `sys-status` binaries on the release tag

## Compile

```
$ go build sys-stats.go
```
## How to use

```sys-status -community [community] -host [host] -port [snmp_port] -interval [interval]```

Default values:
 - community: `public`
 - host: `127.0.0.1`
 - port: `161`
 - interval: `5s`

## Options

Type `$ sys-status -help`

```
Usage: sys-status [options]
  -community string
        community string for snmp (default "public")
  -host string
        hostname or ip address (default "localhost")
  -interval string
        interval in seconds before send another snmp request (default "5s")
  -port uint
        port number (default 161)
  -version
        output version
```

## Dependencies

[gosnmp](https://github.com/soniah/gosnmp) used for snmp requests
```
go get github.com/soniah/gosnmp
```