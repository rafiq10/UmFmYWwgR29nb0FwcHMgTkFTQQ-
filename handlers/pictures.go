package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ErrStr struct {
	Error string `json:"error"`
}

func JSONError(w http.ResponseWriter, err *ErrStr, code int) {

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	e := json.NewEncoder(w).Encode(err)
	if e != nil {
		panic(e)
	}
}

type UrlStore interface {
	GetPictureUrlByDate(from, to string) ([]string, error)
}
type Response struct {
	URLS []string `json:"urls"`
}

type PictureUrlServer struct {
	store UrlStore
}

func (p *PictureUrlServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const dateLayout = "2006-01-02"
	var err error
	var e *ErrStr
	if r.Method != http.MethodGet {
		e = &ErrStr{http.StatusText(http.StatusMethodNotAllowed)}
		JSONError(w, e, http.StatusMethodNotAllowed)
		return
	}

	from := r.FormValue("from")
	if from == "" {
		e = &ErrStr{Error: "'from' has to be provided"}
		JSONError(w, e, http.StatusBadRequest)
		return
	}
	_, err = time.Parse(dateLayout, from)
	if err != nil {
		e = &ErrStr{Error: "'from' argument format not correct\n" + err.Error()}
		JSONError(w, e, http.StatusBadRequest)
		return
	}

	to := r.FormValue("to")
	if to == "" {
		e = &ErrStr{Error: "'to' has to be provided"}
		JSONError(w, e, http.StatusBadRequest)
		return
	}
	_, err = time.Parse(dateLayout, to)
	if err != nil {
		e = &ErrStr{Error: "'to' argument format not correct\n" + err.Error()}
		JSONError(w, e, http.StatusBadRequest)
		return
	}

	if from > to {
		e = &ErrStr{Error: "'to' cannot be earlier than 'from'"}
		JSONError(w, e, http.StatusBadRequest)
		return
	}
	res, err := p.store.GetPictureUrlByDate(from, to)
	if err != nil {
		e = &ErrStr{Error: fmt.Sprintf("p.store.GetPictureUrlByDate(%s, %s)=%v", from, to, err)}
		JSONError(w, e, http.StatusBadRequest)
		return
	}

	if len(res) == 0 {
		e = &ErrStr{Error: http.StatusText(http.StatusNotFound)}
		JSONError(w, e, http.StatusNotFound)
		return
	}

	rsp := Response{URLS: res}
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(rsp)
	if err != nil {
		panic(err)
	}
}
