// Logmon is a server for text files watching. Add some files to watch
// and listen to updates through the socket with a frontend application.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/websocket"
	middleware "github.com/iharsuvorau/http-middleware"
	"github.com/iharsuvorau/logmon"
)

var systemlog = "/var/log/system.log"
var wlist = logmon.NewWatchList()

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	//_ = wlist.Add(systemlog)

	mux := http.DefaultServeMux

	http.HandleFunc("/", handler)
	log.Println("listening at :8080...")
	log.Fatal(http.ListenAndServe(":8080", middleware.Logger(mux)))
}

// / GET list
// / POST a file in JSON: {"filepath": "/foo/bar"}
// /?id=3 GET
// /?id=2 DELETE a file

type response struct {
	Data    interface{} `json:"data,omitempty"`
	ID      uint64      `json:"id,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
	Size    int64       `json:"size,omitempty"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	var err error
	if err = r.ParseForm(); err != nil {
		log.Fatal("failed to parse the URL: " + err.Error())
	}

	id := r.Form.Get("id")
	resp := new(response)

	w.Header().Add("Access-Control-Allow-Origin", "*")

	switch r.Method {
	case "GET":
		if len(id) == 0 {
			// list
			resp.Data = wlist.ListPathes()
			err = json.NewEncoder(w).Encode(resp)
		} else {
			// view
			conn, err := upgrader.Upgrade(w, r, nil)
			defer conn.Close()
			if err != nil {
				log.Println("failed to upgrade: " + err.Error())
				return
			}

			idUint, err := strconv.ParseUint(id, 10, 64)
			if err != nil {
				log.Println("string to uint64 conversion failed: " + err.Error())
				return
			}
			go wlist.Get(idUint).Watch()

			b := []byte{}
			for {
				select {
				case b = <-wlist.Get(idUint).Data:
					if err = conn.WriteMessage(websocket.TextMessage, b); err != nil {
						log.Println("failed to write a message: " + err.Error())
						return
					}
				case err = <-wlist.Get(idUint).Errs:
					log.Println("watching error: " + err.Error())
					return
				}
			}
		}
	case "POST":
		// add
		data := struct {
			Filepath string `json:"filepath"`
		}{}
		if err = json.NewDecoder(r.Body).Decode(&data); err != nil {
			log.Println("failed to decode body: " + err.Error())
		}
		defer r.Body.Close()

		if len(data.Filepath) > 0 {
			stat, err := os.Stat(data.Filepath)
			if err != nil {
				resp.Error = "failed to get file's stat"
				w.WriteHeader(http.StatusBadRequest)
			} else if stat.IsDir() {
				resp.Error = "must be a file, not a directory"
				w.WriteHeader(http.StatusBadRequest)
			} else {
				fID, err := wlist.Add(data.Filepath)
				if err != nil {
					resp.Error = err.Error()
					w.WriteHeader(http.StatusBadRequest)
				} else {
					resp.ID = fID
					resp.Size = stat.Size()
				}
			}
		} else {
			resp.Error = "filepath is missing"
			w.WriteHeader(http.StatusBadRequest)
		}
		err = json.NewEncoder(w).Encode(resp)
	case "DELETE":
		// delete
		idUint, err := strconv.ParseUint(id, 10, 64)
		if err != nil {
			log.Println("string to uint64 conversion failed: " + err.Error())
			return
		}
		if idUint > 0 {
			wlist.Del(idUint)
			resp.Message = "deleted successfully"
		} else {
			resp.Error = "ID must be greater than 0"
		}
		err = json.NewEncoder(w).Encode(resp)
	case "OPTIONS":
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Accept-Language, Content-Language, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
		return
	}

	if err != nil {
		log.Fatal("failed to encode data: " + err.Error())
	}
}
