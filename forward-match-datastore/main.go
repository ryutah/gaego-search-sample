package main

import (
	"encoding/json"
	"net/http"

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

const utf8LastChar = "\xef\xbf\xbd"

func searchSampleDatas(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	var (
		familyName = r.FormValue("familyName")
		givenName  = r.FormValue("givenName")
		email      = r.FormValue("email")
	)

	q := datastore.NewQuery("foo")
	// XXX 比較クエリは複数のプロパティに指定できないため、以下のような検索をするとエラーが発生する
	// http://localhost:8080/foos?familyName=foo&givenName=bar
	if familyName != "" {
		q = q.Filter("FamilyName >=", familyName).Filter("FamilyName <=", familyName+utf8LastChar)
	}
	if givenName != "" {
		q = q.Filter("GivenName >=", givenName).Filter("GivenName <=", givenName+utf8LastChar)
	}
	if email != "" {
		q = q.Filter("Email >=", email).Filter("Email <=", email+utf8LastChar)
	}

	foos := make([]*foo, 0)
	if _, err := q.GetAll(ctx, &foos); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
