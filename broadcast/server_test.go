package broadcast

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"github.com/stretchr/testify/assert"
)

type testServer struct {
	t                  *testing.T
	name               string
	prefix             string
	expectedReqHost    string
	expectedReqHeader  http.Header
	expectedReqBody    []byte
	expectedMethod     string
	expectedRequestURI string
	respStatus         int
	respHeader         http.Header
	respBody           []byte
}

func (d *testServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	assert.Equal(d.t, d.expectedReqHost, r.Host, "request host")
	assert.Equal(d.t, d.expectedMethod, r.Method, "request method")
	assert.Equal(d.t, d.expectedRequestURI, r.RequestURI, fmt.Sprintf("requesturi of %s", d.name))
	for k, v := range d.expectedReqHeader {
		assert.Equal(d.t, v, r.Header[k], fmt.Sprintf("%s for %s", k, d.name))
		r.Header.Del(k)
	}

	expectedReqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		assert.Fail(d.t, err.Error())
	}
	defer r.Body.Close()
	assert.Equal(d.t, d.expectedReqBody, expectedReqBody, fmt.Sprintf("request body of %s", d.name))

	for k, varray := range d.respHeader {
		for _, v := range varray {
			w.Header().Set(k, v)
		}
	}
	w.WriteHeader(d.respStatus)
	w.Write(d.respBody)
}

func TestServer(t *testing.T) {
	// 1a Given values
	httppub := &Server{}
	ms := httptest.NewServer(httppub)
	defer ms.Close()
	msurl, _ := url.Parse(ms.URL)

	givenReqQuery := "k=123&v=456"
	givenRequestURI := "/prefix/path"
	givenReqBody := []byte(`knock knock`)
	givenReqHeader := func() http.Header {
		return http.Header{
			"Accept-Encoding":  []string{"gzip"},
			"Content-Type":     []string{"application/vnd.test"},
			"User-Agent":       []string{"Go-http-client/1.1"},
			"Content-Length":   []string{"11"},
			"X-Forwarded-Host": []string{msurl.Host},
		}
	}

	primary := testServer{
		t:                  t,
		name:               "primary",
		prefix:             "?X-Hello=Some+value",
		expectedReqHeader:  http.Header{"X-Hello": []string{"Some value"}},
		expectedMethod:     "POST",
		expectedRequestURI: givenRequestURI + "?" + givenReqQuery,
		respStatus:         200,
		respHeader:         http.Header{"X-Reply": []string{"who's there?"}},
		respBody:           []byte(`primary`),
	}

	testCases := []*testServer{
		&primary,
		// other targets
		&testServer{
			prefix:             "/server/base?hello=world",
			expectedReqHeader:  http.Header{},
			expectedMethod:     "POST",
			expectedRequestURI: "/server/base" + givenRequestURI + "?" + givenReqQuery,
			respStatus:         201,
			respHeader:         http.Header{"X-Reply": []string{"wrong respHeader"}},
			respBody:           []byte(`wrong respBody`),
		},
		&testServer{
			prefix:             "/server/base?hello=world#fixed",
			expectedReqHeader:  http.Header{},
			expectedMethod:     "POST",
			expectedRequestURI: "/server/base?" + givenReqQuery,
			respStatus:         201,
			respHeader:         http.Header{"X-Reply": []string{"wrong respHeader"}},
			respBody:           []byte(`wrong respBody`),
		},
		&testServer{
			prefix:             "",
			expectedReqHeader:  http.Header{},
			expectedMethod:     "POST",
			expectedRequestURI: givenRequestURI + "?" + givenReqQuery,
			respStatus:         201,
			respHeader:         http.Header{"X-Reply": []string{"wrong respHeader"}},
			respBody:           []byte(`wrong respBody`),
		},
		&testServer{
			prefix: "?X-Forwarded-Host=def.co.uk",
			expectedReqHeader: http.Header{
				"X-Forwarded-Host": []string{"def.co.uk"},
			},
			expectedMethod:     "POST",
			expectedRequestURI: givenRequestURI + "?" + givenReqQuery,
			respStatus:         201,
			respHeader:         http.Header{"X-Reply": []string{"wrong respHeader"}},
			respBody:           []byte(`wrong respBody`),
		},
		&testServer{
			prefix:             "?Host=abc.com",
			expectedReqHost:    "abc.com",
			expectedReqHeader:  http.Header{},
			expectedMethod:     "POST",
			expectedRequestURI: givenRequestURI + "?" + givenReqQuery,
			respStatus:         201,
			respHeader:         http.Header{"X-Reply": []string{"wrong respHeader"}},
			respBody:           []byte(`wrong respBody`),
		},
		&testServer{
			prefix:             "/server/base?Method=PATCH",
			expectedReqHeader:  http.Header{},
			expectedMethod:     "PATCH",
			expectedRequestURI: "/server/base" + givenRequestURI + "?" + givenReqQuery,
			respStatus:         201,
			respHeader:         http.Header{"X-Reply": []string{"wrong respHeader"}},
			respBody:           []byte(`wrong respBody`),
		},
	}

	for i, target := range testCases {
		target.name = fmt.Sprintf("server%d", i)
		target.t = t
		target.expectedReqBody = givenReqBody
		target.expectedReqHeader = mergeHeaders(givenReqHeader(), target.expectedReqHeader)

		server := httptest.NewServer(target)
		targetURL, _ := url.Parse(server.URL + target.prefix)
		defer server.Close()
		if target.expectedReqHost == "" {
			target.expectedReqHost = targetURL.Host
		}
		httppub.Targets = append(httppub.Targets, *targetURL)
	}

	// 1c Given request
	requrl := ms.URL + givenRequestURI + "?" + givenReqQuery
	log.Printf("%#v", requrl)
	req, _ := http.NewRequest("POST", requrl, bytes.NewReader(givenReqBody))
	req.Header = givenReqHeader()
	client := cleanhttp.DefaultClient()

	// 2. Make request to httppub & assert result
	resp, err := client.Do(req)
	assert.Equal(t, primary.respStatus, resp.StatusCode, fmt.Sprintf("resp.StatusCode %#v", err))
	respBody, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	assert.Equal(t, primary.respBody, respBody, fmt.Sprintf("resp.Body %#v", err))
	for k, v := range primary.respHeader {
		assert.Equal(t, v, resp.Header[k], k)
	}
}
