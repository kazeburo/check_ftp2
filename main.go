package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/jlaffaye/ftp"
)

var version string

const UNKNOWN = 3
const CRITICAL = 2
const WARNING = 1
const OK = 0

type commandOpts struct {
	Timeout  time.Duration `long:"timeout" default:"10s" description:"Timeout to wait for connection"`
	Hostname string        `short:"H" long:"hostname" description:"IP address or Host name" default:"127.0.0.1"`
	Port     int           `short:"p" long:"port" description:"Port number" default:"21"`
	SSL      bool          `short:"S" long:"ssl" description:"use TLS"`
	SNI      string        `long:"sni" description:"sepecify hostname for SNI"`
	Explicit bool          `long:"explicit" description:"Use Explicit TLS mode"`
	TCP4     bool          `short:"4" description:"use tcp4 only"`
	TCP6     bool          `short:"6" description:"use tcp6 only"`
	Version  bool          `short:"v" long:"version" description:"Show version"`
}

func dialOptions(opts commandOpts) []ftp.DialOption {
	options := []ftp.DialOption{}

	options = append(options, ftp.DialWithTimeout(opts.Timeout))

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	if opts.SNI != "" {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         opts.SNI,
		}
	}

	if opts.Explicit {
		options = append(options, ftp.DialWithExplicitTLS(tlsConfig))
	}

	dialFunc := func(_, _ string) (net.Conn, error) {
		dialer := &net.Dialer{
			Timeout:   opts.Timeout,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}
		tcpMode := "tcp"
		if opts.TCP4 {
			tcpMode = "tcp4"
		}
		if opts.TCP6 {
			tcpMode = "tcp6"
		}
		conn, err := dialer.Dial(tcpMode, net.JoinHostPort(opts.Hostname, fmt.Sprintf("%d", opts.Port)))
		if err != nil {
			return nil, err
		}
		if opts.SSL && !opts.Explicit {
			conn = tls.Client(conn, tlsConfig)
		}
		return conn, nil
	}
	options = append(options, ftp.DialWithDialFunc(dialFunc))

	return options
}

const replacement = "\\n"

var replacer = strings.NewReplacer(
	"\r\n", replacement,
	"\r", replacement,
	"\n", replacement,
)

func replaceReplacer(s string) string {
	s = strings.TrimRight(s, "\n")
	s = strings.TrimRight(s, "\r")
	return replacer.Replace(s)
}

type reqError struct {
	msg  string
	code int
}

func (e *reqError) Error() string {
	return e.msg
}

func (e *reqError) Code() int {
	return e.code
}
func doConnect(opts commandOpts) (string, *reqError) {

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()
	ch := make(chan error, 1)
	out := ""
	start := time.Now()

	go func() {
		dialopts := dialOptions(opts)
		b := new(bytes.Buffer)
		dialopts = append(dialopts, ftp.DialWithDebugOutput(b))

		c, e := ftp.Dial(
			net.JoinHostPort(opts.Hostname, fmt.Sprintf("%d", opts.Port)),
			dialopts...,
		)
		if e != nil {
			ch <- e
			return
		}
		out = b.String()
		defer c.Quit()
		ch <- nil
	}()

	var err error
	select {
	case err = <-ch:
		// nothing
	case <-ctx.Done():
		err = fmt.Errorf("connection or tls handshake timeout")
	}
	duration := time.Since(start)

	out = replaceReplacer(out)

	if err != nil {
		return "", &reqError{
			fmt.Sprintf("FTP CRITICAL: %v on %s port %d [%s]", err, opts.Hostname, opts.Port, out),
			CRITICAL,
		}

	}

	okMsg := fmt.Sprintf(`FTP OK - %.3f second response time on %s port %d [%s]|time=%fs;;;0.000000;%f`, duration.Seconds(), opts.Hostname, opts.Port, out, duration.Seconds(), opts.Timeout.Seconds())

	return okMsg, nil
}

func printVersion() {
	fmt.Printf(`%s Compiler: %s %s`,
		version,
		runtime.Compiler,
		runtime.Version())
}

func main() {
	os.Exit(_main())
}

func _main() int {
	opts := commandOpts{}
	psr := flags.NewParser(&opts, flags.Default)
	_, err := psr.Parse()
	if err != nil {
		os.Exit(UNKNOWN)
	}

	if opts.Version {
		printVersion()
		return OK
	}

	if opts.TCP4 && opts.TCP6 {
		fmt.Printf("Both tcp4 and tcp6 are specified\n")
		return UNKNOWN
	}

	msg, reqErr := doConnect(opts)
	if reqErr != nil {
		fmt.Println(reqErr.Error())
		return reqErr.Code()
	}
	fmt.Println(msg)
	return OK
}
