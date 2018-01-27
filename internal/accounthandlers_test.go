package internal

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glynternet/go-accounting-storage"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Test_accounts(t *testing.T) {
	t.Run("nil response writer", func(t *testing.T) {
		code, err := accounts(nil, nil)
		assert.Error(t, err)
		assert.Equal(t, http.StatusInternalServerError, code)
	})

	t.Run("NewStorage error", func(t *testing.T) {
		NewStorage = mockStorage{}.newStorageFunc(true)
		rec := httptest.NewRecorder()
		code, err := accounts(rec, nil)
		assert.Error(t, err)
		assert.Equal(t, mockStorageFuncError, errors.Cause(err))
		assert.Equal(t, http.StatusServiceUnavailable, code)
	})

	for _, test := range []struct {
		name string
		code int
		as   *storage.Accounts
		err  error
	}{
		{
			name: "error",
			code: http.StatusServiceUnavailable,
			err:  errors.New("selecting accounts"),
		},
		{
			name: "success",
			code: http.StatusOK,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			NewStorage = mockStorage{Accounts: test.as, err: test.err}.newStorageFunc(false)
			rec := httptest.NewRecorder()
			code, err := accounts(rec, nil)
			assert.Equal(t, test.code, code)

			if test.err != nil {
				assert.Equal(t, test.err, errors.Cause(err))
				return
			}

			assert.NoError(t, err)
			ct := rec.HeaderMap[`Content-Type`]
			assert.Len(t, ct, 1)
			assert.Equal(t, `application/json; charset=UTF-8`, ct[0])
			assert.NoError(t, err)
		})
	}
}
