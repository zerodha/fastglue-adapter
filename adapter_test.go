package fastglueadapter

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func TestNewFastGlueHandler(t *testing.T) {
	expectedMethod := fasthttp.MethodGet
	expectedProto := "HTTP/1.1"
	expectedProtoMajor := 1
	expectedProtoMinor := 1
	expectedRequestURI := "/"
	expectedBody := "body 123 foo bar baz"
	expectedContentLength := len(expectedBody)
	expectedTransferEncoding := "encoding"
	expectedHost := "foobar.com"
	expectedRemoteAddr := "172.217.167.174:80"
	expectedHeader := map[string]string{
		"Foo-Bar":         "baz",
		"Abc":             "defg",
		"XXX-Remote-Addr": "123.43.4543.345",
	}
	expectedURL, err := url.ParseRequestURI(expectedRequestURI)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	expectedContextKey := "contextKey"
	expectedContextValue := "contextValue"

	callsCount := 0
	nethttpH := func(w http.ResponseWriter, r *http.Request) {
		callsCount++
		if r.Method != expectedMethod {
			t.Fatalf("unexpected method %q. Expecting %q", r.Method, expectedMethod)
		}
		if r.Proto != expectedProto {
			t.Fatalf("unexpected proto %q. Expecting %q", r.Proto, expectedProto)
		}
		if r.ProtoMajor != expectedProtoMajor {
			t.Fatalf("unexpected protoMajor %d. Expecting %d", r.ProtoMajor, expectedProtoMajor)
		}
		if r.ProtoMinor != expectedProtoMinor {
			t.Fatalf("unexpected protoMinor %d. Expecting %d", r.ProtoMinor, expectedProtoMinor)
		}
		if r.RequestURI != expectedRequestURI {
			t.Fatalf("unexpected requestURI %q. Expecting %q", r.RequestURI, expectedRequestURI)
		}
		if r.ContentLength != int64(expectedContentLength) {
			t.Fatalf("unexpected contentLength %d. Expecting %d", r.ContentLength, expectedContentLength)
		}
		if len(r.TransferEncoding) != 1 || r.TransferEncoding[0] != expectedTransferEncoding {
			t.Fatalf("unexpected transferEncoding %q. Expecting %q", r.TransferEncoding, expectedTransferEncoding)
		}
		if r.Host != expectedHost {
			t.Fatalf("unexpected host %q. Expecting %q", r.Host, expectedHost)
		}
		if r.RemoteAddr != expectedRemoteAddr {
			t.Fatalf("unexpected remoteAddr %q. Expecting %q", r.RemoteAddr, expectedRemoteAddr)
		}
		body, err := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			t.Fatalf("unexpected error when reading request body: %s", err)
		}
		if string(body) != expectedBody {
			t.Fatalf("unexpected body %q. Expecting %q", body, expectedBody)
		}
		if !reflect.DeepEqual(r.URL, expectedURL) {
			t.Fatalf("unexpected URL: %#v. Expecting %#v", r.URL, expectedURL)
		}
		if r.Context().Value(expectedContextKey) != expectedContextValue {
			t.Fatalf("unexpected context value for key %q. Expecting %q", expectedContextKey, expectedContextValue)
		}

		for k, expectedV := range expectedHeader {
			v := r.Header.Get(k)
			if v != expectedV {
				t.Fatalf("unexpected header value %q for key %q. Expecting %q", v, k, expectedV)
			}
		}

		w.Header().Set("Header1", "value1")
		w.Header().Set("Header2", "value2")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "request body is %q", body)
	}
	fasglueH := NewFastGlueHandler(http.HandlerFunc(nethttpH))
	fastglueH := setContextValueMiddleware(fasglueH, expectedContextKey, expectedContextValue)

	req := fastglue.Request{}
	req.RequestCtx = &fasthttp.RequestCtx{}

	req.RequestCtx.Request.Header.SetMethod(expectedMethod)
	req.RequestCtx.Request.SetRequestURI(expectedRequestURI)
	req.RequestCtx.Request.Header.SetHost(expectedHost)
	req.RequestCtx.Request.Header.Add(fasthttp.HeaderTransferEncoding, expectedTransferEncoding)
	if _, err := req.RequestCtx.Request.BodyWriter().Write([]byte(expectedBody)); err != nil {
		t.Fatalf("failed to set body")
	}
	for k, v := range expectedHeader {
		req.RequestCtx.Request.Header.Set(k, v)
	}

	remoteAddr, err := net.Dial("tcp", expectedRemoteAddr)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	req.RequestCtx.Init2(remoteAddr, nil, true) //(&req.RequestCtx.Request, remoteAddr, nil)

	err = fastglueH(&req)
	if err != nil {
		t.Fatalf("[fastglueH] unexpected error: %s", err)
	}
	if callsCount != 1 {
		t.Fatalf("unexpected callsCount: %d. Expecting 1", callsCount)
	}

	resp := &req.RequestCtx.Response
	if resp.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("unexpected statusCode: %d. Expecting %d", resp.StatusCode(), fasthttp.StatusBadRequest)
	}
	if string(resp.Header.Peek("Header1")) != "value1" {
		t.Fatalf("unexpected header value: %q. Expecting %q", resp.Header.Peek("Header1"), "value1")
	}
	if string(resp.Header.Peek("Header2")) != "value2" {
		t.Fatalf("unexpected header value: %q. Expecting %q", resp.Header.Peek("Header2"), "value2")
	}
	expectedResponseBody := fmt.Sprintf("request body is %q", expectedBody)
	if string(resp.Body()) != expectedResponseBody {
		t.Fatalf("unexpected response body %q. Expecting %q", resp.Body(), expectedResponseBody)
	}
}

func setContextValueMiddleware(next fastglue.FastRequestHandler, key string, value interface{}) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		r.RequestCtx.SetUserValue(key, value)
		return next(r)
	}
}
