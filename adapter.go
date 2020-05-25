// Package fastglueadapter provides helper functions for converting net/http
// request handlers to fastglue request handlers.
package fastglueadapter

import (
	"io"
	"net/http"
	"net/url"

	"github.com/zerodha/fastglue"
)

// NewFastGlueHandlerFunc wraps net/http handler func to fastglue
// request handler, so it can be passed to fastglue server.
//
// While this function may be used for easy switching from net/http to fastglue,
// it has the following drawbacks comparing to using manually written fastglue
// request handler:
//
//     * A lot of useful functionality provided by fastglue is missing
//       from net/http handler such as webhooks.
//     * net/http -> fastglue handler conversion has some overhead,
//       so the returned handler will be always slower than manually written
//       fastglue handler.
//
// So it is advisable using this function only for quick net/http -> fastglue
// switching. Then manually convert net/http handlers to fastglue handlers
// according to https://github.com/valyala/fasthttp#switching-from-nethttp-to-fasthttp .
func NewFastGlueHandlerFunc(h http.HandlerFunc) fastglue.FastRequestHandler {
	return NewFastGlueHandler(h)
}

// NewFastGlueHandler wraps net/http handler to fastglue request handler,
// so it can be passed to fastglue server.
//
// While this function may be used for easy switching from net/http to fastglue,
// it has the following drawbacks comparing to using manually written fastglue
// request handler:
//
//     * A lot of useful functionality provided by fastglue is missing
//       from net/http handler.
//     * net/http -> fastglue handler conversion has some overhead,
//       so the returned handler will be always slower than manually written
//       fastglue handler.
//
// So it is advisable using this function only for quick net/http -> fastglue
// switching. Then manually convert net/http handlers to fastglue handlers
// according to https://github.com/valyala/fastglue#switching-from-nethttp-to-fasthttp .
func NewFastGlueHandler(h http.Handler) fastglue.FastRequestHandler {
	return func(req *fastglue.Request) error {
		var r http.Request

		body := req.RequestCtx.PostBody()
		r.Method = string(req.RequestCtx.Method())
		r.Proto = "HTTP/1.1"
		r.ProtoMajor = 1
		r.ProtoMinor = 1
		r.RequestURI = string(req.RequestCtx.RequestURI())
		r.ContentLength = int64(len(body))
		r.Host = string(req.RequestCtx.Host())
		r.RemoteAddr = req.RequestCtx.RemoteAddr().String()

		hdr := make(http.Header)
		req.RequestCtx.Request.Header.VisitAll(func(k, v []byte) {
			sk := string(k)
			sv := string(v)
			switch sk {
			case "Transfer-Encoding":
				r.TransferEncoding = append(r.TransferEncoding, sv)
			default:
				hdr.Set(sk, sv)
			}
		})
		r.Header = hdr
		r.Body = &netHTTPBody{body}
		rURL, err := url.ParseRequestURI(r.RequestURI)
		if err != nil {
			req.RequestCtx.Logger().Printf("cannot parse requestURI %q: %s", r.RequestURI, err)
			req.RequestCtx.Error("Internal Server Error", http.StatusInternalServerError)
			return err
		}
		r.URL = rURL

		var w netHTTPResponseWriter
		h.ServeHTTP(&w, r.WithContext(req.RequestCtx))

		req.RequestCtx.SetStatusCode(w.StatusCode())
		for k, vv := range w.Header() {
			for _, v := range vv {
				req.RequestCtx.Response.Header.Set(k, v)
			}
		}
		_, err = req.RequestCtx.Write(w.body)
		return err
	}

}

type netHTTPBody struct {
	b []byte
}

func (r *netHTTPBody) Read(p []byte) (int, error) {
	if len(r.b) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.b)
	r.b = r.b[n:]
	return n, nil
}

func (r *netHTTPBody) Close() error {
	r.b = r.b[:0]
	return nil
}

type netHTTPResponseWriter struct {
	statusCode int
	h          http.Header
	body       []byte
}

func (w *netHTTPResponseWriter) StatusCode() int {
	if w.statusCode == 0 {
		return http.StatusOK
	}
	return w.statusCode
}

func (w *netHTTPResponseWriter) Header() http.Header {
	if w.h == nil {
		w.h = make(http.Header)
	}
	return w.h
}

func (w *netHTTPResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (w *netHTTPResponseWriter) Write(p []byte) (int, error) {
	w.body = append(w.body, p...)
	return len(p), nil
}
