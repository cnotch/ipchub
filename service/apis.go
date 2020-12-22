// Copyright (c) 2019,CAOHONGJU All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package service

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cnotch/apirouter"
	"github.com/cnotch/ipchub/config"
	"github.com/cnotch/ipchub/media"
	"github.com/cnotch/ipchub/provider/auth"
	"github.com/cnotch/ipchub/provider/route"
	"github.com/cnotch/ipchub/stats"
)

const (
	usernameHeaderKey = "user_name_in_token"
)

var (
	buffers = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, 1024*2))
		},
	}
	noAuthRequired = map[string]bool{
		"/api/v1/login":        true,
		"/api/v1/server":       true,
		"/api/v1/runtime":      true,
		"/api/v1/refreshtoken": true,
	}
)

var crossdomainxml = []byte(
	`<?xml version="1.0" ?><cross-domain-policy>
			<allow-access-from domain="*" />
			<allow-http-request-headers-from domain="*" headers="*"/>
		</cross-domain-policy>`)

func (s *Service) initApis(mux *http.ServeMux) {
	api := apirouter.NewForGRPC(
		// 系统信息类API
		apirouter.POST("/api/v1/login", s.onLogin),
		apirouter.GET("/api/v1/server", s.onGetServerInfo),
		apirouter.GET("/api/v1/runtime", s.onGetRuntime),
		apirouter.GET("/api/v1/refreshtoken", s.onRefreshToken),

		// 流管理API
		apirouter.GET("/api/v1/streams", s.onListStreams),
		apirouter.GET("/api/v1/streams/{path=**}", s.onGetStreamInfo),
		apirouter.DELETE("/api/v1/streams/{path=**}", s.onStopStream),
		apirouter.DELETE("/api/v1/streams/{path=**}:consumer", s.onStopConsumer),

		// 路由管理API
		apirouter.GET("/api/v1/routes", s.onListRoutes),
		apirouter.GET("/api/v1/routes/{pattern=**}", s.onGetRoute),
		apirouter.DELETE("/api/v1/routes/{pattern=**}", s.onDelRoute),
		apirouter.POST("/api/v1/routes", s.onSaveRoute),

		// 用户管理API
		apirouter.GET("/api/v1/users", s.onListUsers),
		apirouter.GET("/api/v1/users/{userName=*}", s.onGetUser),
		apirouter.DELETE("/api/v1/users/{userName=*}", s.onDelUser),
		apirouter.POST("/api/v1/users", s.onSaveUser),
	)

	iterc := apirouter.ChainInterceptor(apirouter.PreInterceptor(s.authInterceptor),
		apirouter.PreInterceptor(roleInterceptor))

	// api add to mux
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		if path.Base(r.URL.Path) == "crossdomain.xml" {
			w.Header().Set("Content-Type", "application/xml")
			w.Write(crossdomainxml)
			return
		}

		path := strings.ToLower(r.URL.Path)
		if _, ok := noAuthRequired[path]; ok || iterc.PreHandle(w, r) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			api.ServeHTTP(w, r)
		}
	})
}

// 刷新Token
func (s *Service) onRefreshToken(w http.ResponseWriter, r *http.Request, pathParams apirouter.Params) {
	token := r.URL.Query().Get("token")
	if token != "" {
		newtoken := s.tokens.Refresh(token)
		if newtoken != nil {
			if err := jsonTo(w, newtoken); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}

	http.Error(w, "Token is not valid", http.StatusUnauthorized)
	return
}

// 登录
func (s *Service) onLogin(w http.ResponseWriter, r *http.Request, pathParams apirouter.Params) {
	type UserCredentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	// 提取凭证
	var uc UserCredentials
	err := json.NewDecoder(r.Body).Decode(&uc)
	if err != nil {
		// 尝试 Form解析
		uc.Username = r.FormValue("username")
		uc.Password = r.FormValue("password")
		if len(uc.Username) == 0 || len(uc.Password) == 0 {
			http.Error(w, "用户名或密码错误", http.StatusForbidden)
			return
		}
	}

	// 验证用户和密码
	u := auth.Get(uc.Username)
	if u == nil || u.ValidatePassword(uc.Password) != nil {
		http.Error(w, "用户名或密码错误", http.StatusForbidden)
		return
	}

	// 新建Token，并返回
	token := s.tokens.NewToken(u.Name)

	if err := jsonTo(w, token); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// 获取运行时信息
func (s *Service) onGetServerInfo(w http.ResponseWriter, r *http.Request, pathParams apirouter.Params) {
	type server struct {
		Vendor   string `json:"vendor"`
		Name     string `json:"name"`
		Version  string `json:"version"`
		OS       string `json:"os"`
		Arch     string `json:"arch"`
		StartOn  string `json:"start_on"`
		Duration string `json:"duration"`
	}
	srv := server{
		Vendor:   config.Vendor,
		Name:     config.Name,
		Version:  config.Version,
		OS:       strings.Title(runtime.GOOS),
		Arch:     strings.ToUpper(runtime.GOARCH),
		StartOn:  stats.StartingTime.Format(time.RFC3339Nano),
		Duration: time.Now().Sub(stats.StartingTime).String(),
	}

	if err := jsonTo(w, &srv); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// 获取运行时信息
func (s *Service) onGetRuntime(w http.ResponseWriter, r *http.Request, pathParams apirouter.Params) {
	const extraKey = "extra"

	type sccc struct {
		SC int `json:"sources"`
		CC int `json:"consumers"`
	}
	type runtime struct {
		On      string            `json:"on"`
		Proc    stats.Proc        `json:"proc"`
		Streams sccc              `json:"streams"`
		Rtsp    stats.ConnsSample `json:"rtsp"`
		Wsp     stats.ConnsSample `json:"wsp"`
		Flv     stats.ConnsSample `json:"flv"`
		Extra   *stats.Runtime    `json:"extra,omitempty"`
	}
	sc, cc := media.Count()

	rt := runtime{
		On:      time.Now().Format(time.RFC3339Nano),
		Proc:    stats.MeasureRuntime(),
		Streams: sccc{sc, cc},
		Rtsp:    stats.RtspConns.GetSample(),
		Wsp:     stats.WspConns.GetSample(),
		Flv:     stats.FlvConns.GetSample(),
	}

	params := r.URL.Query()
	if strings.TrimSpace(params.Get(extraKey)) == "1" {
		rt.Extra = stats.MeasureFullRuntime()
	}

	if err := jsonTo(w, &rt); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Service) onListStreams(w http.ResponseWriter, r *http.Request, pathParams apirouter.Params) {
	params := r.URL.Query()
	pageSize, pageToken, err := listParamers(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	includeCS := strings.TrimSpace(params.Get("c")) == "1"

	count, sinfos := media.Infos(pageToken, pageSize, includeCS)
	type streamInfos struct {
		Total         int                 `json:"total"`
		NextPageToken string              `json:"next_page_token"`
		Streams       []*media.StreamInfo `json:"streams,omitempty"`
	}

	list := &streamInfos{
		Total:   count,
		Streams: sinfos,
	}
	if len(sinfos) > 0 {
		list.NextPageToken = sinfos[len(sinfos)-1].Path
	}

	if err := jsonTo(w, list); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Service) onGetStreamInfo(w http.ResponseWriter, r *http.Request, pathParams apirouter.Params) {
	path := pathParams.ByName("path")

	var rt *media.Stream

	rt = media.Get(path)
	if rt == nil {
		http.NotFound(w, r)
		return
	}

	params := r.URL.Query()
	includeCS := strings.TrimSpace(params.Get("c")) == "1"

	si := rt.Info(includeCS)

	if err := jsonTo(w, si); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Service) onStopStream(w http.ResponseWriter, r *http.Request, pathParams apirouter.Params) {
	path := pathParams.ByName("path")

	var rt *media.Stream

	rt = media.Get(path)
	if rt != nil {
		rt.Close()
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Service) onStopConsumer(w http.ResponseWriter, r *http.Request, pathParams apirouter.Params) {
	path := pathParams.ByName("path")
	param := r.URL.Query().Get("cid")
	no, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var rt *media.Stream
	rt = media.Get(path)
	if rt != nil {
		rt.StopConsume(media.CID(no))
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Service) onListRoutes(w http.ResponseWriter, r *http.Request, pathParams apirouter.Params) {
	params := r.URL.Query()
	pageSize, pageToken, err := listParamers(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	routes := route.All()
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Pattern < routes[j].Pattern
	})

	begini := 0
	for _, r1 := range routes {
		if r1.Pattern <= pageToken {
			begini++
			continue
		}
		break
	}

	type routeInfos struct {
		Total         int            `json:"total"`
		NextPageToken string         `json:"next_page_token"`
		Routes        []*route.Route `json:"routes,omitempty"`
	}

	list := &routeInfos{
		Total:         len(routes),
		NextPageToken: pageToken,
		Routes:        make([]*route.Route, 0, pageSize),
	}

	j := 0
	for i := begini; i < len(routes) && j < pageSize; i++ {
		j++
		list.Routes = append(list.Routes, routes[i])
		list.NextPageToken = routes[i].Pattern
	}

	if err := jsonTo(w, list); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Service) onGetRoute(w http.ResponseWriter, r *http.Request, pathParams apirouter.Params) {
	pattern := pathParams.ByName("pattern")
	r1 := route.Get(pattern)
	if r1 == nil {
		http.NotFound(w, r)
		return
	}

	if err := jsonTo(w, r1); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Service) onDelRoute(w http.ResponseWriter, r *http.Request, pathParams apirouter.Params) {
	pattern := pathParams.ByName("pattern")
	err := route.Del(pattern)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Service) onSaveRoute(w http.ResponseWriter, r *http.Request, pathParams apirouter.Params) {
	r1 := &route.Route{}
	err := json.NewDecoder(r.Body).Decode(r1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = route.Save(r1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Service) onListUsers(w http.ResponseWriter, r *http.Request, pathParams apirouter.Params) {
	params := r.URL.Query()
	pageSize, pageToken, err := listParamers(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	users := auth.All()
	sort.Slice(users, func(i, j int) bool {
		return users[i].Name < users[j].Name
	})

	begini := 0
	for _, u := range users {
		if u.Name <= pageToken {
			begini++
			continue
		}
		break
	}

	type userInfos struct {
		Total         int         `json:"total"`
		NextPageToken string      `json:"next_page_token"`
		Users         []auth.User `json:"users,omitempty"`
	}

	list := &userInfos{
		Total:         len(users),
		NextPageToken: pageToken,
		Users:         make([]auth.User, 0, pageSize),
	}

	j := 0
	for i := begini; i < len(users) && j < pageSize; i++ {
		j++
		u := *users[i]
		u.Password = ""
		list.Users = append(list.Users, u)
		list.NextPageToken = u.Name
	}

	if err := jsonTo(w, list); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Service) onGetUser(w http.ResponseWriter, r *http.Request, pathParams apirouter.Params) {
	userName := pathParams.ByName("userName")
	u := auth.Get(userName)
	if u == nil {
		http.NotFound(w, r)
		return
	}

	u2 := *u
	u2.Password = ""
	if err := jsonTo(w, &u2); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Service) onDelUser(w http.ResponseWriter, r *http.Request, pathParams apirouter.Params) {
	userName := pathParams.ByName("userName")
	err := auth.Del(userName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Service) onSaveUser(w http.ResponseWriter, r *http.Request, pathParams apirouter.Params) {
	u := &auth.User{}
	err := json.NewDecoder(r.Body).Decode(u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	updatePassword := r.URL.Query().Get("update_password") == "1"
	err = auth.Save(u, updatePassword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func jsonTo(w io.Writer, o interface{}) error {
	formatted := buffers.Get().(*bytes.Buffer)
	formatted.Reset()
	defer buffers.Put(formatted)

	body, err := json.Marshal(o)
	if err != nil {
		return err
	}

	if err := json.Indent(formatted, body, "", "\t"); err != nil {
		return err
	}

	if _, err := w.Write(formatted.Bytes()); err != nil {
		return err
	}
	return nil
}

func listParamers(params url.Values) (pageSize int, pageToken string, err error) {
	pageSizeStr := params.Get("page_size")
	pageSize = 20
	if pageSizeStr != "" {
		var err error
		pageSize, err = strconv.Atoi(pageSizeStr)
		if err != nil {
			return pageSize, pageToken, err
		}
	}
	pageToken = params.Get("page_token")
	return
}

// ?token=
func (s *Service) authInterceptor(w http.ResponseWriter, r *http.Request) bool {
	token := r.URL.Query().Get("token")
	if token != "" {
		username := s.tokens.AccessCheck(token)
		if username != "" {
			r.Header.Set(usernameHeaderKey, username)
			return true // 继续执行
		}
	}

	http.Error(w, "Token is not valid", http.StatusUnauthorized)
	return false
}

func roleInterceptor(w http.ResponseWriter, r *http.Request) bool {
	// 流查询方法，无需管理员身份
	if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/streams") {
		return true
	}

	userName := r.Header.Get(usernameHeaderKey)
	u := auth.Get(userName)
	if u == nil || !u.Admin {
		http.Error(w /*http.StatusText(http.StatusForbidden)*/, "访问被拒绝，请用管理员登录", http.StatusForbidden)
		return false
	}

	return true
}
