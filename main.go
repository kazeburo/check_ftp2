package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
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

type Opt struct {
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

func (o *Opt) dialOptions() []ftp.DialOption {
	options := []ftp.DialOption{}

	options = append(options, ftp.DialWithTimeout(o.Timeout))

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	if o.SNI != "" {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         o.SNI,
		}
	}

	if o.Explicit {
		options = append(options, ftp.DialWithExplicitTLS(tlsConfig))
	}

	dialFunc := func(_, _ string) (net.Conn, error) {
		dialer := &net.Dialer{
			Timeout:   o.Timeout,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}
		tcpMode := "tcp"
		if o.TCP4 {
			tcpMode = "tcp4"
		}
		if o.TCP6 {
			tcpMode = "tcp6"
		}
		conn, err := dialer.Dial(tcpMode, net.JoinHostPort(o.Hostname, fmt.Sprintf("%d", o.Port)))
		if err != nil {
			return nil, err
		}
		if o.SSL && !o.Explicit {
			tlsconn := tls.Client(conn, tlsConfig)
			err = tlsconn.Handshake()
			if err != nil {
				return nil, err
			}
			return tlsconn, nil
		}
		return conn, nil
	}
	options = append(options, ftp.DialWithDialFunc(dialFunc))

	return options
}

type ftpError struct {
	msg  string
	code int
}

func (e *ftpError) Error() string {
	return e.msg
}

func (e *ftpError) Code() int {
	return e.code
}

type ftpConnectResult struct {
	out string
	err error
}

func (o *Opt) doConnect() (string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), o.Timeout)
	defer cancel()
	ch := make(chan ftpConnectResult, 1)
	start := time.Now()

	go func() {
		dialopts := o.dialOptions()
		b := new(bytes.Buffer)
		dialopts = append(dialopts, ftp.DialWithDebugOutput(b))

		c, e := ftp.Dial(
			net.JoinHostPort(o.Hostname, fmt.Sprintf("%d", o.Port)),
			dialopts...,
		)
		if e != nil {
			ch <- ftpConnectResult{"", e}
			return
		}
		defer c.Quit()
		ch <- ftpConnectResult{b.String(), nil}
	}()

	var res ftpConnectResult
	select {
	case res = <-ch:
		// nothing
	case <-ctx.Done():
		res = ftpConnectResult{
			out: "",
			err: fmt.Errorf("connection or tls handshake timeout"),
		}
	}
	duration := time.Since(start)

	res.out = replaceReplacer(res.out)

	if res.err != nil {
		return "", &ftpError{
			msg:  fmt.Sprintf("FTP CRITICAL: %v on %s port %d [%s]", res.err, o.Hostname, o.Port, res.out),
			code: CRITICAL,
		}

	}

	okMsg := fmt.Sprintf(`FTP OK - %.3f second response time on %s port %d [%s]|time=%fs;;;0.000000;%f`, duration.Seconds(), o.Hostname, o.Port, res.out, duration.Seconds(), o.Timeout.Seconds())

	return okMsg, nil
}

func main() {
	os.Exit(_main())
}

func _main() int {
	opt := &Opt{}
	psr := flags.NewParser(opt, flags.HelpFlag|flags.PassDoubleDash)
	_, err := psr.Parse()
	if opt.Version {
		fmt.Printf(`%s %s
Compiler: %s %s
`,
			os.Args[0],
			version,
			runtime.Compiler,
			runtime.Version())
		return OK
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}

	if opt.TCP4 && opt.TCP6 {
		fmt.Printf("Both tcp4 and tcp6 are specified\n")
		return UNKNOWN
	}

	msg, err := opt.doConnect()
	if err != nil {
		var ftpErr *ftpError
		switch {
		case errors.As(err, &ftpErr):
			fmt.Println(ftpErr.Error())
			return ftpErr.Code()
		default:
			fmt.Printf("FTP connection failed with unexpected error: %v\n", err)
			return CRITICAL
		}
	}

	fmt.Println(msg)
	return OK
}
