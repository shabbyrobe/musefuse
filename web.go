package musefuse

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/NYTimes/gziphandler"
	service "github.com/shabbyrobe/go-service"
	"github.com/shabbyrobe/go-service/serviceutil"
	"github.com/shabbyrobe/golib/httptools"
)

type WebServer struct {
	host   string
	server *http.Server
	fs     *FS
}

func NewWebServer(host string, fs *FS) *WebServer {
	srv := &WebServer{
		host: host,
		fs:   fs,
	}

	router := http.NewServeMux()
	router.Handle("/", &indexHandler{fs: fs})

	routerCORS := httptools.CORSHandler{Handler: router}
	routerGz := gziphandler.GzipHandler(routerCORS)
	srv.server = &http.Server{
		Handler: routerGz,
		Addr:    host,
	}

	return srv
}

func (s *WebServer) Run(ctx service.Context) error {
	svc := serviceutil.NewHTTP(s.server)
	return svc.Run(ctx)
}

type indexHandler struct {
	fs *FS
}

func (index *indexHandler) ServeHTTP(rs http.ResponseWriter, rq *http.Request) {
	node := index.fs.lookup(rq.URL.Path)
	if node == nil {
		http.NotFound(rs, rq)
		return
	}

	if dir, ok := node.(*dirNode); ok {
		for _, entry := range dir.entries {
			fmt.Fprintln(rs, entry.Name)
		}

	} else if file, ok := node.(*fileNode); ok {
		bts, err := json.MarshalIndent(file.entry, "", "  ")
		if err != nil {
			http.Error(rs, "data marshal failed", 500)
			return
		}
		rs.Write(bts)
	}
}
