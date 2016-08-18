package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/xjdrew/go-udtwrapper/udt"
)

var udtaddr = flag.String("udtaddr", "", "Remote UDT address (host:port) to connect.")
var udtport = flag.Int("udtport", 0, "Local UDT port to listen.")

var tcpaddr = flag.String("tcpaddr", "", "Remote TCP address (host:port) to connect.")
var tcpport = flag.Int("tcpport", 0, "Local TCP port to listen.")

var bufsize = flag.Int("bufsize", 1024*1024, "Send/receive buffer size.")

var stopc chan struct{}
var stopped bool = false

func exitOnError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(-1)
	}
}

type Dialer interface {
	Dial(network, address string) (net.Conn, error)
}

func main() {
	flag.Parse()

	if flag.NFlag() == 0 {
		flag.Usage()
		os.Exit(0)
	}

	if *udtaddr != "" && *tcpaddr != "" {
		fmt.Fprintf(os.Stderr, "both udt and tcp address shouldn't be specified.\n")
		os.Exit(0)
	}

	if *udtaddr == "" && *tcpaddr == "" {
		fmt.Fprintf(os.Stderr, "either udt or tcp address should be specified.\n")
		os.Exit(0)
	}

	if *udtaddr != "" && *tcpport == 0 {
		fmt.Fprintf(os.Stderr, "tcp port should be specified for listening.\n")
		os.Exit(0)
	}

	if *tcpaddr != "" && *udtport == 0 {
		fmt.Fprintf(os.Stderr, "udt port should be specified for listening.\n")
		os.Exit(0)
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	stopc = make(chan struct{}, 1)
	go func() {
		<-sigc
		stopped = true
		close(stopc)
	}()

	var ln net.Listener
	var d Dialer
	var err error
	var network, raddr string

	if *udtaddr != "" {
		fmt.Printf("listening on tcp port %d...\n", *tcpport)
		ln, err = net.Listen("tcp", fmt.Sprintf(":%d", *tcpport))
		exitOnError(err)

		d = &udt.Dialer{}
		network = "udt"
		raddr = *udtaddr
	} else {
		fmt.Printf("listening on udt port %d...\n", *tcpport)
		ln, err = udt.Listen("udt", fmt.Sprintf(":%d", *udtport))
		exitOnError(err)

		d = &net.Dialer{}
		network = "tcp"
		raddr = *tcpaddr
	}

	go func() {
		<-stopc
		ln.Close()
	}()

	for {
		fmt.Printf("waiting for local connection...\n")
		c, err := ln.Accept()
		if stopped {
			break
		}
		exitOnError(err)

		fmt.Printf("new local connection from %s.\n", c.RemoteAddr())
		go handle(d, network, raddr, c)
	}
}

func handle(d Dialer, network, raddr string, l net.Conn) {
	fmt.Printf("connecting to remote address %s...\n", raddr)
	r, err := d.Dial(network, raddr)
	exitOnError(err)

	fmt.Printf("tunneling between %s and %s...\n", l.RemoteAddr(), r.RemoteAddr())
	tunnel(l, r)

	l.Close()
	r.Close()
}

func tunnel(l, r net.Conn) {
	donec := make(chan struct{}, 2)

	go func() {
		buf := make([]byte, *bufsize)
		n := 0
		for {
			nr, rerr := r.Read(buf)
			if rerr != nil && rerr != io.EOF {
				fmt.Fprintf(os.Stderr, "reading from %s failed: %s\n", r.RemoteAddr(), rerr.Error())
				break
			}

			fmt.Printf("read %d bytes from remote\n", nr)

			if nr > 0 {
				nw, werr := l.Write(buf[:nr])
				if werr != nil {
					fmt.Fprintf(os.Stderr, "writing to %s failed: %s\n", l.RemoteAddr(), werr.Error())
					break
				}

				fmt.Printf("written %d bytes to local\n", nw)
				n += nw
			}

			if rerr == io.EOF {
				break
			}
		}

		fmt.Printf("tunnel %s<->%s received %d bytes.\n", l.RemoteAddr(), r.RemoteAddr(), n)
		donec <- struct{}{}
	}()

	go func() {
		buf := make([]byte, *bufsize)
		n := 0
		for {
			nr, rerr := l.Read(buf)
			if rerr != nil && rerr != io.EOF {
				fmt.Fprintf(os.Stderr, "reading from %s failed: %s\n", l.RemoteAddr(), rerr.Error())
				break
			}

			fmt.Printf("read %d bytes from local\n", nr)

			if nr > 0 {
				nw, werr := r.Write(buf[:nr])
				if werr != nil {
					fmt.Fprintf(os.Stderr, "writing %s remote failed: %s\n", r.RemoteAddr(), werr.Error())
					break
				}

				fmt.Printf("written %d bytes to remote\n", nw)
				n += nw
			}

			if rerr == io.EOF {
				break
			}
		}

		fmt.Printf("tunnel %s<->%s sent %d bytes.\n", l.RemoteAddr(), r.RemoteAddr(), n)
		donec <- struct{}{}
	}()

	select {
	case <-donec:
	case <-stopc:
		return
	}
}
