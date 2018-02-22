package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/gorilla/mux"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
)

type foo struct {
	FamilyName string   `datastore:",noindex"`
	GivenName  string   `datastore:",noindex"`
	Email      string   `datastore:",noindex"`
	Search     []string `json:"-"`
}

func (f *foo) createBiGram() []string {
	// 文字列のトークナイズ 簡単のためBigramのみ生成
	var (
		family = nGram(f.FamilyName, 2, "*", "f")
		given  = nGram(f.GivenName, 2, "*", "g")
		email  = nGram(f.Email, 2, "*", "e")
	)

	index := make([]string, 0, len(family)+len(given)+len(email))
	index = append(index, family...)
	index = append(index, given...)
	index = append(index, email...)

	return index
}

func (f *foo) Load(property []datastore.Property) error {
	for _, p := range property {
		switch p.Name {
		case "FamilyName":
			f.FamilyName = p.Value.(string)
		case "GivenName":
			f.GivenName = p.Value.(string)
		case "Email":
			f.Email = p.Value.(string)
		}
	}
	return nil
}

func (f *foo) Save() ([]datastore.Property, error) {
	p := []datastore.Property{
		datastore.Property{
			Name:    "FamilyName",
			Value:   f.FamilyName,
			NoIndex: true,
		},
		datastore.Property{
			Name:    "GivenName",
			Value:   f.GivenName,
			NoIndex: true,
		},
		datastore.Property{
			Name:    "Email",
			Value:   f.Email,
			NoIndex: true,
		},
	}
	grams := f.createBiGram()
	for _, g := range grams {
		prop := datastore.Property{
			Name:     "Search",
			Value:    g,
			Multiple: true,
		}
		p = append(p, prop)
	}
	return p, nil
}

func init() {
	r := mux.NewRouter()

	r.HandleFunc("/foos", searchSampleDatas).Methods(http.MethodGet)
	r.HandleFunc("/foos", putSampleDatas).Methods(http.MethodPost)

	http.Handle("/", r)
}

func searchSampleDatas(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	query := r.FormValue("q")
	filter := nGram(query, 2, "*")
	q := datastore.NewQuery("foo2")
	for _, f := range filter {
		q = q.Filter("Search=", f)
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
		foo{FamilyName: "メロン", GivenName: "太郎", Email: "meron@sample.com"},
		foo{FamilyName: "ロンメロ", GivenName: "太郎", Email: "ronmero@sample.com"},
	}

	keys := []*datastore.Key{
		datastore.NewIncompleteKey(ctx, "foo2", nil),
		datastore.NewIncompleteKey(ctx, "foo2", nil),
		datastore.NewIncompleteKey(ctx, "foo2", nil),
		datastore.NewIncompleteKey(ctx, "foo2", nil),
		datastore.NewIncompleteKey(ctx, "foo2", nil),
		datastore.NewIncompleteKey(ctx, "foo2", nil),
		datastore.NewIncompleteKey(ctx, "foo2", nil),
		datastore.NewIncompleteKey(ctx, "foo2", nil),
	}

	if _, err := datastore.PutMulti(ctx, keys, foos); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func nGram(str string, n int, prefix ...string) []string {
	var (
		newstr  = str
		size    = 0
		runeidx = make([]int, 1, len(str))
	)

	for len(newstr) > 0 {
		_, wide := utf8.DecodeRuneInString(newstr)
		size += wide
		runeidx = append(runeidx, size)
		newstr = newstr[wide:]
	}

	ret := make([]string, 0, len(str)*(len(prefix)+1))
	for i, j := 0, n; j < len(runeidx); j++ {
		left, right := runeidx[i], runeidx[j]
		s := str[left:right]
		for _, p := range prefix {
			ret = append(ret, fmt.Sprintf("%s %s", p, s))
		}
		i = j - (n - 1)
	}

	return ret
}
