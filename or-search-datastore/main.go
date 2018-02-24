package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/mux"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
)

type foo struct {
	FamilyName string
	GivenName  string
	Email      string
}

func init() {
	r := mux.NewRouter()

	r.HandleFunc("/foos", searchSampleDatas).Methods(http.MethodGet)
	r.HandleFunc("/foos", putSampleDatas).Methods(http.MethodPost)

	http.Handle("/", r)
}

func searchSampleDatas(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	var (
		familyName = r.FormValue("familyName")
		givenName  = r.FormValue("givenName")
		email      = r.FormValue("email")
	)

	var (
		wg   = new(sync.WaitGroup)
		mux  = new(sync.Mutex)
		foos []*foo
		errs []error
	)

	getAll := func(q *datastore.Query, mux *sync.Mutex) {
		var f []*foo
		_, err := q.GetAll(ctx, &f)

		mux.Lock()
		defer mux.Unlock()
		if err == nil {
			foos = append(foos, f...)
		} else {
			errs = append(errs, err)
		}
	}
	q := datastore.NewQuery("foo")

	// 検索パラメータが指定されていた場合は検索ワードをフィルタリング条件として並列で検索を行う
	if familyName != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			getAll(q.Filter("FamilyName=", familyName), mux)
		}()
	}
	if givenName != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			getAll(q.Filter("GivenName=", givenName), mux)
		}()
	}
	if email != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			getAll(q.Filter("Email=", email), mux)
		}()
	}
	wg.Wait()

	if len(errs) != 0 {
		http.Error(w, fmt.Sprintf("%v", errs), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	body, _ := json.MarshalIndent(foos, "", "  ")
	w.Write(body)
}

func putSampleDatas(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	foos := []foo{
		foo{FamilyName: "田中", GivenName: "太郎", Email: "tanaka@sample.com"},
		foo{FamilyName: "田所", GivenName: "三郎", Email: "tadokoro@sample.com"},
		foo{FamilyName: "鈴木", GivenName: "一郎", Email: "i-suzuki@sample.com"},
		foo{FamilyName: "鈴木", GivenName: "次郎", Email: "j-tanaka@sample.com"},
		foo{FamilyName: "山田", GivenName: "花子", Email: "h-yamada@sample.com"},
		foo{FamilyName: "山田", GivenName: "太郎", Email: "t-yamada@sample.com"},
	}

	keys := []*datastore.Key{
		datastore.NewIncompleteKey(ctx, "foo", nil),
		datastore.NewIncompleteKey(ctx, "foo", nil),
		datastore.NewIncompleteKey(ctx, "foo", nil),
		datastore.NewIncompleteKey(ctx, "foo", nil),
		datastore.NewIncompleteKey(ctx, "foo", nil),
		datastore.NewIncompleteKey(ctx, "foo", nil),
	}

	if _, err := datastore.PutMulti(ctx, keys, foos); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
