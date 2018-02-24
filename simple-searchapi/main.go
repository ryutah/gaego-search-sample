package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gorilla/mux"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/search"
	"google.golang.org/appengine/taskqueue"
)

type foo struct {
	FamilyName string
	GivenName  string
	Email      string
}

type fooIndex struct {
	FamilyName string
	GivenName  string
	Email      string
}

func init() {
	r := mux.NewRouter()

	r.HandleFunc("/foos", searchSampleDatas).Methods(http.MethodGet)
	r.HandleFunc("/foos", putSampleDatas).Methods(http.MethodPost)

	r.HandleFunc("/backend/foos/index", createFooIndex).Methods(http.MethodPost)

	http.Handle("/", r)
}

func searchSampleDatas(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	// 検索ワードの取得
	q := r.FormValue("q")
	index, err := search.Open("foo")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Search APIで検索を行う
	// Search APIは検索インデックスとしての用途のみ期待しており、実データはDatastoreから取得するようにするため、
	// 検索オプションとしてIDsOnlyを指定している。
	iterator := index.Search(ctx, q, &search.SearchOptions{
		IDsOnly: true,
	})
	var (
		iteError error
		keys     []*datastore.Key
	)
	// 検索結果の取得
	for {
		sid, err := iterator.Next(nil)
		if err == search.Done {
			break
		} else if err != nil {
			iteError = err
			break
		}
		id, _ := strconv.ParseInt(sid, 10, 64)
		keys = append(keys, datastore.NewKey(ctx, "foo", "", id, nil))
	}
	if iteError != nil {
		http.Error(w, iteError.Error(), http.StatusInternalServerError)
		return
	}

	// Search APIの検索結果のIDをもとに、Datastoreから実データを取得する
	foos := make([]*foo, len(keys))
	if err := datastore.GetMulti(ctx, keys, foos); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	body, _ := json.MarshalIndent(foos, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func putSampleDatas(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	// サンプルデータの投入
	foos := []foo{
		foo{FamilyName: "田中", GivenName: "太郎", Email: "tanaka@sample.com"},
		foo{FamilyName: "田所", GivenName: "三郎", Email: "tadokoro@sample.com"},
		foo{FamilyName: "鈴木", GivenName: "一郎", Email: "i-suzuki@sample.com"},
		foo{FamilyName: "鈴木", GivenName: "次郎", Email: "j-tanaka@sample.com"},
		foo{FamilyName: "山田", GivenName: "花子", Email: "h-yamada@sample.com"},
		foo{FamilyName: "テストユーザー", GivenName: "ほげ太郎", Email: "tanaka@sample.com"},
		foo{FamilyName: "sample users", GivenName: "foo user", Email: "sample@sample.com"},
	}

	keys := []*datastore.Key{
		datastore.NewIncompleteKey(ctx, "foo", nil),
		datastore.NewIncompleteKey(ctx, "foo", nil),
		datastore.NewIncompleteKey(ctx, "foo", nil),
		datastore.NewIncompleteKey(ctx, "foo", nil),
		datastore.NewIncompleteKey(ctx, "foo", nil),
		datastore.NewIncompleteKey(ctx, "foo", nil),
		datastore.NewIncompleteKey(ctx, "foo", nil),
	}

	newKeys, err := datastore.PutMulti(ctx, keys, foos)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 検索インデックスの作成タスクを行う。
	// リクエストのレイテンシを下げるために、インデックスの作成はTaskqueueを利用してバックグラウンドで行うようにしている
	tasks := make([]*taskqueue.Task, 0, len(newKeys))
	for _, key := range newKeys {
		val := url.Values{"id": {strconv.FormatInt(key.IntID(), 10)}}
		t := taskqueue.NewPOSTTask("/backend/foos/index", val)
		tasks = append(tasks, t)
	}
	if _, err := taskqueue.AddMulti(ctx, tasks, "default"); err != nil {
		log.Errorf(ctx, "failed to create index create task")
	}

	w.WriteHeader(http.StatusCreated)
}

func createFooIndex(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	// リクエストボディからSearch APIインデックス構築対象となるエンティティを取得してくる
	sid := r.FormValue("id")
	id, err := strconv.ParseInt(sid, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Search APIインデックス構築対象のエンティティをDatastoreから取得する
	var (
		key = datastore.NewKey(ctx, "foo", "", id, nil)
		foo = new(foo)
	)
	if err := datastore.Get(ctx, key, foo); err != nil {
		log.Errorf(ctx, "failed to get foo; id: %v, error: %#v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Search APIインデックス構築処理
	index, err := search.Open("foo")
	if err != nil {
		log.Errorf(ctx, "failed to open index foo : %#v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fooIdx := &fooIndex{
		FamilyName: foo.FamilyName,
		GivenName:  foo.GivenName,
		Email:      foo.Email,
	}
	// Datastoreと紐付けるために、Search APIのインデックスのIDでとして、DatastoreのエンティティのIDを指定している
	if _, err := index.Put(ctx, strconv.FormatInt(id, 10), fooIdx); err != nil {
		log.Errorf(ctx, "failed to put index : %#v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
