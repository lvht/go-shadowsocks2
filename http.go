package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/textproto"
	"net/url"
	"strings"

	"github.com/shadowsocks/go-shadowsocks2/socks"
)

func getReq(c net.Conn) (req bytes.Buffer, host string, isHTTPS bool, err error) {
	r := bufio.NewReader(c)
	tp := textproto.NewReader(r)

	// First line: GET /index.html HTTP/1.0
	var requestLine string
	if requestLine, err = tp.ReadLine(); err != nil {
		return
	}

	method, requestURI, _, ok := parseRequestLine(requestLine)
	if !ok {
		err = fmt.Errorf("malformed HTTP request")
		return
	}

	// https request
	if method == "CONNECT" {
		isHTTPS = true
		requestURI = "http://" + requestURI
	}

	// get remote host
	uriInfo, err := url.ParseRequestURI(requestURI)
	if err != nil {
		return
	}

	// Subsequent lines: Key: value.
	headers, err := tp.ReadMIMEHeader()
	if err != nil {
		return
	}

	if uriInfo.Host == "" {
		host = headers.Get("Host")
	} else {
		if strings.Index(uriInfo.Host, ":") == -1 {
			host = uriInfo.Host + ":80"
		} else {
			host = uriInfo.Host
		}
	}

	req.WriteString(requestLine + "\r\n")
	for k, vs := range headers {
		for _, v := range vs {
			req.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
		}
	}
	req.WriteString("\r\n")
	if r.Buffered() > 0 {
		r.WriteTo(&req)
	}
	return
}

func parseRequestLine(line string) (method, requestURI, proto string, ok bool) {
	s1 := strings.Index(line, " ")
	s2 := strings.Index(line[s1+1:], " ")
	if s1 < 0 || s2 < 0 {
		return
	}
	s2 += s1 + 1
	return line[:s1], line[s1+1 : s2], line[s2+1:], true
}

func httpLocal(addr, server string, shadow func(net.Conn) net.Conn) {
	defer func() { c2 <- nil }()

	logf("HTTP proxy %s <-> %s", addr, server)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		logf(err.Error())
		return
	}

	for {
		c, err := l.Accept()
		if err != nil {
			logf("[http] failed to accept: %s", err)
			continue
		}

		go func() {
			defer c.Close()
			req, target, isHTTPS, err := getReq(c)
			if err != nil {
				logf(err.Error())
				return
			}

			logf("[http] connecting to " + target)
			err = dialss(c, target, server, shadow, func(rc net.Conn, isDirect bool) error {
				addr := socks.ParseAddr(target)
				if addr == nil {
					return fmt.Errorf("invalid addr: %s", target)
				}

				if !isDirect {
					logf("[http] use socks connect to " + addr.String())
					if _, err := rc.Write(addr); err != nil {
						logf(err.Error())
						return err
					}
				}

				if isHTTPS {
					_, err = c.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
					if err != nil {
						logf(err.Error())
						return err
					}
				} else {
					_, err := req.WriteTo(rc)
					if err != nil {
						logf(err.Error())
						return err
					}
				}

				return nil
			})

			if err != nil {
				logf(err.Error())
			}

			return
		}()
	}
}
