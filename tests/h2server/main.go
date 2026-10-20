// Independent HTTP/2 peer for tarn-http2 integration: Go's net/http
// unencrypted HTTP/2 (prior knowledge). Listens on 127.0.0.1:0 and prints the
// address on its first line.
package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func pattern(i int) byte { return byte(i % 251) }

func main() {
	mux := http.NewServeMux()
	// Full duplex: echo each body chunk as it arrives, then a trailer.
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != 2 || r.Header.Get("x-client") != "tarn" {
			http.Error(w, "bad request", 400)
			return
		}
		w.Header().Set("Trailer", "X-Total")
		// Long, repeated values: Go's encoder uses Huffman and the dynamic table.
		w.Header().Set("X-Echo-Note", strings.Repeat("streaming duplex audio ", 8))
		w.WriteHeader(200)
		w.(http.Flusher).Flush()
		total := 0
		buf := make([]byte, 32768)
		for {
			n, err := r.Body.Read(buf)
			if n > 0 {
				total += n
				w.Write(buf[:n])
				w.(http.Flusher).Flush()
			}
			if err != nil {
				break
			}
		}
		w.Header().Set("X-Total", strconv.Itoa(total))
	})
	mux.HandleFunc("/large", func(w http.ResponseWriter, r *http.Request) {
		n, _ := strconv.Atoi(r.URL.Query().Get("n"))
		out := make([]byte, n)
		for i := range out {
			out[i] = pattern(i)
		}
		w.Write(out)
	})
	mux.HandleFunc("/count", func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		for i, b := range data {
			if b != pattern(i) {
				http.Error(w, "corrupt at "+strconv.Itoa(i), 500)
				return
			}
		}
		w.Header().Set("X-Count", strconv.Itoa(len(data)))
		w.WriteHeader(204)
	})
	mux.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	})
	mux.HandleFunc("/headers", func(w http.ResponseWriter, r *http.Request) {
		for i := 0; i < 40; i++ {
			w.Header().Set(fmt.Sprintf("X-Field-%02d", i), strings.Repeat("v", 20+i))
		}
		w.WriteHeader(200)
	})
	protocols := new(http.Protocols)
	protocols.SetUnencryptedHTTP2(true)
	server := &http.Server{Handler: mux, Protocols: protocols}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(listener.Addr().String())
	server.Serve(listener)
}
