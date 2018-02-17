package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"io/ioutil"

	"github.com/glynternet/go-accounting-storage"
	account2 "github.com/glynternet/go-accounting/account"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

func (s *server) handlerSelectAccounts(w http.ResponseWriter, _ *http.Request) (int, error) {
	if w == nil {
		return http.StatusInternalServerError, errors.New("nil ResponseWriter")
	}
	store, err := s.NewStorage()
	if err != nil {
		return http.StatusServiceUnavailable, errors.Wrap(err, "creating new storage")
	}
	var as *storage.Accounts
	as, err = store.SelectAccounts()
	if err != nil {
		return http.StatusServiceUnavailable, errors.Wrap(err, "selecting handlerSelectAccounts from client")
	}
	w.Header().Set(`Content-Type`, `application/json; charset=UTF-8`)
	return http.StatusOK, errors.Wrap(
		json.NewEncoder(w).Encode(as),
		"error encoding handlerSelectAccounts json",
	)
}

func (s *server) muxAccountIDHandlerFunc(w http.ResponseWriter, r *http.Request) (int, error) {
	vars := mux.Vars(r)
	if vars == nil {
		return http.StatusBadRequest, errors.New("no context variables")
	}

	key := "id"
	idString, ok := vars[key]
	if !ok {
		return http.StatusBadRequest, errors.New("no account_id context variable")
	}
	id, err := strconv.ParseUint(idString, 10, 64)
	if err != nil {
		return http.StatusBadRequest, errors.Wrapf(err, "parsing %s to uint", key)
	}
	return s.handlerSelectAccount(uint(id))(w, r)
}

func (s *server) handlerSelectAccount(id uint) appHandler {
	return func(w http.ResponseWriter, r *http.Request) (int, error) {
		if w == nil {
			return http.StatusInternalServerError, errors.New("nil ResponseWriter")
		}
		store, err := s.NewStorage()
		if err != nil {
			return http.StatusServiceUnavailable, errors.Wrap(err, "creating new storage")
		}
		var a *storage.Account
		a, err = store.SelectAccount(id)
		if err != nil {
			return http.StatusNotFound, errors.Wrap(err, "selecting handlerSelectAccount from storage")
		}
		w.Header().Set(`Content-Type`, `application/json; charset=UTF-8`)
		return http.StatusOK, errors.Wrap(
			json.NewEncoder(w).Encode(a),
			"error encoding handlerSelectAccounts json",
		)
	}
}

///// THIS NEEDS SPLITTING UP AND ISN'T GOING SO WELL AT THE MOMENT
///// I THINK IT WOULD BE BEST TO SPLIT UP INTO SOMETHING THAT TAKES IN A REQUEST
///// AND PARSES IT INTO AN ACCOUNT, THEN WE CAN JUST HAVE A HANDLER THAT TAKES
///// AN ACCOUNT AND WRITES TO A RESPONSE WRITER???

func (s *server) handlerInsertAccount(w http.ResponseWriter, r *http.Request) (int, error) {
	if w == nil {
		return http.StatusInternalServerError, errors.New("nil ResponseWriter")
	}
	store, err := s.NewStorage()
	if err != nil {
		return http.StatusServiceUnavailable, errors.Wrap(err, "creating new storage")
	}
	bod, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return http.StatusBadRequest, errors.Wrap(err, "reading all body")
	}
	inner := new(account2.Account)
	err = json.Unmarshal(bod, inner)
	if err != nil {
		return http.StatusBadRequest, errors.Wrap(err, "unmarshalling body to account")
	}
	a, err := store.InsertAccount(*inner)
	if err != nil {
		return http.StatusBadRequest, errors.Wrap(err, "inserting account to storage")
	}
	w.Header().Set(`Content-Type`, `application/json; charset=UTF-8`)
	return http.StatusOK, errors.Wrap(
		json.NewEncoder(w).Encode(a),
		"error encoding handlerSelectAccounts json",
	)
}
