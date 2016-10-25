/*
 GNU GENERAL PUBLIC LICENSE
                       Version 3, 29 June 2007

 Copyright (C) 2007 Free Software Foundation, Inc. <http://fsf.org/>
 Everyone is permitted to copy and distribute verbatim copies
 of this license document, but changing it is not allowed.*/

package brainiac

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/npotts/go-patterns/stoppable"
	"github.com/pkg/errors"
	"github.com/tylerb/graceful"
	"github.com/urfave/negroni"
	"net/http"
	"time"
)

var _ = fmt.Println

type httpd struct {
	httpd            *graceful.Server //stoppable server
	mux              *mux.Router      //http router
	negroni          *negroni.Negroni //middelware
	regFxn, storeFxn regstore         //callback fxns
	user             string           //http info
	pass             string           //password
	stopper          stoppable.Halter //atomic halter
}

func newHTTP(cfg Config, reg, store regstore) (*httpd, error) {
	err := make(chan error)
	defer close(err)
	neg := negroni.Classic()
	h := &httpd{
		stopper: stoppable.NewStopable(),
		mux:     mux.NewRouter(),
		negroni: neg,
		httpd: &graceful.Server{
			Timeout: 100 * time.Millisecond, //no timeout, which has its own set of issues
			Server: &http.Server{
				Addr:           cfg.HTTPListen,
				Handler:        neg,
				ReadTimeout:    1 * time.Second,
				WriteTimeout:   1 * time.Second,
				MaxHeaderBytes: 1024 * 1024 * 1024 * 10, //10meg
			},
		},
		regFxn:   reg,
		storeFxn: store,
		user:     cfg.HTTPUser,
		pass:     cfg.HTTPPassword,
	}
	h.mux.HandleFunc("/{table:[a-zA-Z]*}", h.put).Methods("PUT")
	h.mux.HandleFunc("/{table:[a-zA-Z]*}", h.post).Methods("POST")
	h.negroni.UseFunc(h.auth)
	h.negroni.UseHandler(h.mux)

	go h.monitor(err)
	return h, <-err
}

/*monitor starts the HTTP server and attempts to keep it going*/
func (h *httpd) monitor(startup chan error) {
	ecc := make(chan error)
	go func() { ecc <- h.httpd.ListenAndServe(); close(ecc) }() //start daemon

	select {
	case <-time.After(100 * time.Millisecond):
		startup <- nil
	case e := <-ecc:
		startup <- e
	}
}

/*stop kills the service*/
func (h *httpd) stop() {
	defer h.stopper.Die()
	if h.stopper.Alive() {
		c := h.httpd.StopChan()
		go func() { h.httpd.Stop(100 * time.Millisecond) }()
		<-c
	}
	return
}

/*fill in with basic authentication validator*/
func (h *httpd) auth(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if u, p, ok := r.BasicAuth(); ok && u == h.user && p == h.pass {
		next(w, r)
		return
	}
	w.WriteHeader(http.StatusUnauthorized)
}

/*handleJSON breaks up json data*/
func (h *httpd) handleJSON(r *http.Request, fxn regstore) error {
	data := make([]byte, r.ContentLength)
	if n, err := r.Body.Read(data); int64(n) != r.ContentLength || err != nil {
		return errors.New("Invalid HTTP data")
	}

	m := Datam{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	if !m.Valid() {
		return errors.New("Invalid HTTP data, didnt populated m")
	}

	return fxn(mux.Vars(r)["table"], m)
}

/*put handles incoming data formats to register*/
func (h *httpd) put(w http.ResponseWriter, r *http.Request) {
	if err := h.handleJSON(r, h.regFxn); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

/*post handles 'inserting' actual data*/
func (h *httpd) post(w http.ResponseWriter, r *http.Request) {
	if err := h.handleJSON(r, h.storeFxn); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}
