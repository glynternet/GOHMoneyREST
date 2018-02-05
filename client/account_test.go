package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"fmt"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestGetAccountsFromEndpoint(t *testing.T) {
	t.Run("get error", func(t *testing.T) {
		c := Client("bloopybloop")
		as, err := c.getAccountsFromEndpoint("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "getting from endpoint")
		assert.Nil(t, as)
	})

	t.Run("unexpected status", func(t *testing.T) {
		srv := newJSONTestServer(nil, http.StatusTeapot)
		defer srv.Close()
		c := Client(srv.URL)
		as, err := c.getAccountsFromEndpoint("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server returned unexpected code")
		assert.Nil(t, as)
	})

	t.Run("unmarshallable response", func(t *testing.T) {
		srv := newJSONTestServer(
			struct{ NonAccount string }{NonAccount: "bloop"},
			http.StatusOK,
		)
		defer srv.Close()
		c := Client(srv.URL)
		as, err := c.getAccountsFromEndpoint("")
		if assert.Error(t, err) {
			assert.IsType(t, &json.UnmarshalTypeError{}, errors.Cause(err))
		}
		assert.Nil(t, as)
		srv.Close()
	})
}

func TestGetAccountFromEndpoint(t *testing.T) {
	t.Run("get error", func(t *testing.T) {
		c := Client("bloopybleep")
		a, err := c.getAccountFromEndpoint("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "getting from endpoint")
		assert.Nil(t, a)
	})

	t.Run("unexpected status", func(t *testing.T) {
		srv := newJSONTestServer(nil, http.StatusTeapot)
		defer srv.Close()
		c := Client(srv.URL)
		as, err := c.getAccountFromEndpoint("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server returned unexpected code")
		assert.Nil(t, as)
	})

	t.Run("unmarshallable response", func(t *testing.T) {
		srv := newJSONTestServer(
			struct{ NonAccount string }{NonAccount: "bloop"},
			http.StatusOK,
		)
		defer srv.Close()
		c := Client(srv.URL)
		as, err := c.getAccountFromEndpoint("")
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "unmarshalling response")
		}
		assert.Nil(t, as)
		srv.Close()
	})
}

func newJSONTestServer(encode interface{}, code int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bs, err := json.Marshal(encode)
		if err != nil {
			panic(fmt.Sprintf("error marshalling to json: %v", err))
		}
		w.WriteHeader(code)
		w.Header().Set(`Content-Type`, `application/json; charset=UTF-8`)
		_, err = w.Write(bs)
		if err != nil {
			panic(fmt.Sprintf("error writing to ResponseWriter: %v", err))
		}
		return
	}))
}