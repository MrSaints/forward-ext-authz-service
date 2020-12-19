// Parts of this software was derived from https://github.com/traefik/traefik/blob/master/pkg/middlewares/auth/forward.go
// As such, The MIT License (MIT) applies.

package main

import (
	"context"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/vulcand/oxy/forward"
	"github.com/vulcand/oxy/utils"
	"go.uber.org/zap"
)

const (
	xForwardedURI    = "X-Forwarded-Uri"
	xForwardedMethod = "X-Forwarded-Method"
)

var (
	forwardAuthRequestTimeout time.Duration = 15 * time.Second
)

type checker interface {
	Check(ctx context.Context, req *Request) (*Response, error)
}

type forwardAuthChecker struct {
	logger *zap.Logger

	// The authentication server address.
	forwardAuthAddress string
	// A list of the headers to copy from the request to the authentication server.
	authRequestHeaders []string
	// A list of the headers to copy from the authentication server to the request.
	authResponseHeaders []string
	// Trust all the existing X-Forwarded-* headers.
	trustForwardHeader bool
}

// Modified from https://github.com/traefik/traefik/blob/a3327c4430cf4a2cb35216776c2c9843281dec00/pkg/middlewares/auth/forward.go#L87
func (f *forwardAuthChecker) Check(ctx context.Context, req *Request) (*Response, error) {
	f.logger.Info("Handling request", zap.String("url", req.Request.URL.String()))

	forwardReq, err := http.NewRequest(http.MethodGet, f.forwardAuthAddress, nil)
	if err != nil {
		return nil, err
	}

	writeHeader(&req.Request, forwardReq, f.trustForwardHeader, f.authRequestHeaders)

	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: forwardAuthRequestTimeout,
	}

	f.logger.Debug("Making forward authN request", zap.String("url", forwardReq.URL.String()), zap.Any("headers", forwardReq.Header))
	forwardRes, err := client.Do(forwardReq)
	if err != nil {
		return nil, err
	}

	resHeaders := http.Header{}
	utils.CopyHeaders(resHeaders, forwardRes.Header)

	if forwardRes.StatusCode < http.StatusOK || forwardRes.StatusCode >= http.StatusMultipleChoices {
		utils.RemoveHeaders(resHeaders, forward.HopHeaders...)

		redirectURL, err := forwardRes.Location()
		if err != nil {
			if !errors.Is(err, http.ErrNoLocation) {
				return nil, err
			}
		} else if redirectURL.String() != "" {
			resHeaders.Set("Location", redirectURL.String())
		}

		f.logger.Debug("Forward auth disallow", zap.Int("statusCode", forwardRes.StatusCode), zap.Any("headers", resHeaders))

		return &Response{
			Allow: false,
			Response: http.Response{
				StatusCode: forwardRes.StatusCode,
				Header:     resHeaders,
			},
		}, nil
	}

	f.logger.Debug("Forward auth allow", zap.Int("statusCode", forwardRes.StatusCode), zap.Any("headers", resHeaders))

	for _, headerName := range f.authResponseHeaders {
		headerKey := http.CanonicalHeaderKey(headerName)
		resHeaders.Del(headerKey)
		if len(forwardRes.Header[headerKey]) > 0 {
			resHeaders[headerKey] = append([]string(nil), forwardRes.Header[headerKey]...)
		}
	}

	return &Response{
		Allow: true,
		Response: http.Response{
			StatusCode: http.StatusOK,
			Header:     resHeaders,
		},
	}, nil
}

// Modified from https://github.com/traefik/traefik/blob/3140a4e0cd8d904a7eadd67255b025c9d842823d/pkg/middlewares/auth/forward.go#L188
func writeHeader(req, forwardReq *http.Request, trustForwardHeader bool, allowedHeaders []string) {
	utils.CopyHeaders(forwardReq.Header, req.Header)
	utils.RemoveHeaders(forwardReq.Header, forward.HopHeaders...)

	forwardReq.Header = filterForwardRequestHeaders(forwardReq.Header, allowedHeaders)

	xMethod := req.Header.Get(xForwardedMethod)
	switch {
	case xMethod != "" && trustForwardHeader:
		forwardReq.Header.Set(xForwardedMethod, xMethod)
	case req.Method != "":
		forwardReq.Header.Set(xForwardedMethod, req.Method)
	default:
		forwardReq.Header.Del(xForwardedMethod)
	}

	xfp := req.Header.Get(forward.XForwardedProto)
	switch {
	case xfp != "" && trustForwardHeader:
		forwardReq.Header.Set(forward.XForwardedProto, xfp)
	case req.TLS != nil:
		forwardReq.Header.Set(forward.XForwardedProto, "https")
	case req.URL != nil && req.URL.Scheme != "":
		forwardReq.Header.Set(forward.XForwardedProto, req.URL.Scheme)
	default:
		forwardReq.Header.Set(forward.XForwardedProto, "http")
	}

	if xfp := req.Header.Get(forward.XForwardedPort); xfp != "" && trustForwardHeader {
		forwardReq.Header.Set(forward.XForwardedPort, xfp)
	}

	xfh := req.Header.Get(forward.XForwardedHost)
	switch {
	case xfh != "" && trustForwardHeader:
		forwardReq.Header.Set(forward.XForwardedHost, xfh)
	case req.Host != "":
		forwardReq.Header.Set(forward.XForwardedHost, req.Host)
	default:
		forwardReq.Header.Del(forward.XForwardedHost)
	}

	xfURI := req.Header.Get(xForwardedURI)
	switch {
	case xfURI != "" && trustForwardHeader:
		forwardReq.Header.Set(xForwardedURI, xfURI)
	case req.URL.RequestURI() != "":
		forwardReq.Header.Set(xForwardedURI, req.URL.RequestURI())
	default:
		forwardReq.Header.Del(xForwardedURI)
	}
}

// Taken from https://github.com/traefik/traefik/blob/3140a4e0cd8d904a7eadd67255b025c9d842823d/pkg/middlewares/auth/forward.go#L248
func filterForwardRequestHeaders(forwardRequestHeaders http.Header, allowedHeaders []string) http.Header {
	if len(allowedHeaders) == 0 {
		return forwardRequestHeaders
	}

	filteredHeaders := http.Header{}
	for _, headerName := range allowedHeaders {
		values := forwardRequestHeaders.Values(headerName)
		if len(values) > 0 {
			filteredHeaders[http.CanonicalHeaderKey(headerName)] = append([]string(nil), values...)
		}
	}

	return filteredHeaders
}
