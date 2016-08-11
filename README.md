# udt-tunnel

Tunnels TCP connection through UDT connection and vice-versa. It can also be used to connect an TCP connection to UDT and vice-versa.

## Tunneling TCP through UDT

On local machine

```
$ udt-tunnel -udtaddr <remote-udt-server-addr:port> -tcpport <local-tcp-listen-port>
```

On remote machine

```
$ udt-tunnel -udtport <local-udt-list-port> -tcpport <local-tcp-server-addr:port>
```

## Connecting TCP with UDT

```
$ udt-tunnel -udtaddr <remote-udt-server-addr:port> -tcpport <local-tcp-listen-port>
```

Now any data sent through the above local TCP port will be sent to the UDT server through UDT protocol.
