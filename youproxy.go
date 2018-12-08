package main

import (
	"flag"
	"fmt"
	"github.com/dragonsinth/slither/main/slitherserver/impl"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"
)

func main() {
	addr := flag.String("addr", ":8080", "http service address")
	flag.Parse()
	if runtime.GOOS == "darwin" {
		register("")
	} else {
		impl.Register("slither.dragonsinth.com")
		register("y.dragonsinth.com")
	}

	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func register(hostname string) {
	http.HandleFunc(hostname+"/", func(w http.ResponseWriter, r *http.Request) {
		v := r.URL.Query()
		vid := v.Get("v")
		if vid == "" {
			http.NotFound(w, r)
			return
		}

		rsp, err := http.Get("https://www.youtube.com/get_video_info?video_id=" + url.QueryEscape(vid))
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get: %v", err), http.StatusInternalServerError)
			return
		}
		if rsp.StatusCode < 200 || rsp.StatusCode > 299 {
			http.Error(w, "proxy fetch failed", rsp.StatusCode)
			return
		}

		body, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to read: %v", err), http.StatusInternalServerError)
			return
		}

		vals, err := url.ParseQuery(string(body))
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to parse: %v", err), http.StatusInternalServerError)
			return
		}
		for k, v := range vals {
			log.Printf("%s: %s", k, v)
		}

		u := ""
		streams := append(vals["url_encoded_fmt_stream_map"], vals["adaptive_fmts"]...)
		for _, s := range streams {
			log.Println()
			log.Printf("%+v", s)
			svals, err := url.ParseQuery(s)
			if err != nil {
				log.Printf("failed to svals: %v", err)
				continue
			}
			itags := svals["itag"]
			urls := svals["url"]
			for i := range itags {
				switch itags[i] {
				case "139":
					u = urls[i]
					break
				case "140":
					u = urls[i]
					break
				case "141":
					u = urls[i]
					break
				}
			}
		}

		if u == "" {
			http.Error(w, fmt.Sprintf("no valid streams: %v", err), http.StatusBadRequest)
			return
		}

		title := vals.Get("title")
		thumbnail := vals.Get("thumbnail_url")
		log.Printf("serving %s (%s): %s", vid, title, u)
		httpGet(fmt.Sprintf(tmpl, html.EscapeString(title), html.EscapeString(u), html.EscapeString(thumbnail)), "text/html; charset=utf-8", w, r)
	})
}

const tmpl = `
<html>
	<head>
		<title>%s</title>
		<meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, minimum-scale=1, user-scalable=no, minimal-ui">
	</head
	<body style="zoom: 200%%;">
		<audio style="width: 100%%;" src="%s" autoplay controls></audio>
		<img style="width: 100%%;" src="%s"></img>
	</body>
</html>
`

func httpGet(data string, contentType string, w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		httpError(w, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Add("Content-Type", contentType)
	http.ServeContent(w, r, "", time.Now(), strings.NewReader(data))
}

func httpError(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}
