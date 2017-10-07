package broadcast

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	cleanhttp "github.com/hashicorp/go-cleanhttp"
)

// Server proxies to many target
type Server struct {
	RequestTimeout time.Duration
	Targets        []url.URL
	wg             sync.WaitGroup
	counter        int64
}

// CleanupWait blocks until all requests are done
// e.g. some non-primary targets may be super slow, and we're still waiting
// for them to complete before we can delete tmpfile
func (h *Server) CleanupWait() {
	h.wg.Wait()
}

// ServeHTTP is http
func (h *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestid := uuid.New().String()
	for {
		if atomic.CompareAndSwapInt64(&h.counter, h.counter, h.counter+1) {
			break
		}
	}
	log.Printf(requestid+": waitGroup %d", h.counter)
	h.wg.Add(1)
	publishRequest(requestid, h.Targets, h.RequestTimeout, w, r, func() {
		// called once when the last target is done
		log.Printf(requestid + ": all done")
		h.wg.Done()
		for {
			if atomic.CompareAndSwapInt64(&h.counter, h.counter, h.counter-1) {
				break
			}
		}
		log.Printf(requestid+": waitGroup %d", h.counter)
	})
}

func publishRequest(requestid string, targets []url.URL, timeout time.Duration,
	w http.ResponseWriter, r *http.Request, onAllDone func()) {
	start := time.Now()
	defer func() {
		log.Printf(requestid+": took %07.4fs", time.Since(start).Seconds())
	}()

	// save request payload to file (for replay)
	tmpfile := path.Join(os.TempDir(), requestid)
	log.Printf(requestid+": \t[cache]   %s", tmpfile)
	wf, err := os.Create(tmpfile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		onAllDone()
		return
	}
	defer r.Body.Close()
	io.Copy(wf, r.Body)
	err = wf.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		onAllDone()
		return
	}

	tmpfileWg := sync.WaitGroup{}
	defer func() {
		// remove `tmpfile` only when all targets are done; `tmpfileWg`
		go func() {
			tmpfileWg.Wait()
			log.Printf(requestid+": \t[delete after %s] %s", time.Since(start), tmpfile)
			os.Remove(tmpfile)
			onAllDone()
		}()
	}()

	replyWg := sync.WaitGroup{}
	for i, target := range targets {
		i := i
		primary := (i == 0)
		target := target

		if primary {
			// only wait for `primary` before we `exit` this `ServeHTTP`
			replyWg.Add(1)
		}
		// wait for ALL targets before we delete `tmpfile`
		tmpfileWg.Add(1)

		go func() {
			statusCode := http.StatusInternalServerError
			if primary {
				defer replyWg.Done()
			}
			defer tmpfileWg.Done()

			resultPath := path.Join(target.Path, r.URL.Path)
			if target.Fragment != "" {
				resultPath = target.Path
			}

			// for each target, determine the target url
			resultURL := target.ResolveReference(&url.URL{
				Path:     resultPath,
				RawQuery: r.URL.RawQuery,
			})

			resultMethod := r.Method
			if m := target.Query().Get("Method"); m != "" {
				resultMethod = m
			}

			var err error
			defer func() {
				log.Printf(requestid+": \t[%d took] %07.4fs - %d - %s %s (%#v)", i, time.Since(start).Seconds(), statusCode, resultMethod, resultURL.String(), err)
			}()

			log.Printf(requestid+": %s %s", resultMethod, resultURL)
			file, err := os.Open(tmpfile)
			if err != nil {
				statusCode = http.StatusBadGateway
				if primary {
					http.Error(w, err.Error(), statusCode)
				}
				return
			}
			defer file.Close()

			// send request to target url with content of file as payload
			req, err := http.NewRequest(resultMethod, resultURL.String(), file)
			if err != nil {
				statusCode = http.StatusBadRequest
				if primary {
					http.Error(w, "Bad request", statusCode)
				}
				return
			}

			// copy request headers
			mergeHeaders(req.Header, r.Header)
			req.ContentLength = r.ContentLength

			// overwritable headers
			for k, varray := range target.Query() {
				for _, v := range varray {
					switch k {
					case "Host":
						req.Host = v
					default:
						req.Header.Set(k, v)
					}
				}
			}

			client := cleanhttp.DefaultClient()
			client.Timeout = timeout
			resp, err := client.Do(req)
			if err != nil {
				statusCode = http.StatusBadGateway
				if primary {
					http.Error(w, "Bad gateway", statusCode)
				}
				return
			}
			defer resp.Body.Close()
			statusCode = resp.StatusCode

			if !primary {
				// if we're NOT the primary, keep quiet
				return
			}

			// relay primary http response
			mergeHeaders(w.Header(), resp.Header)
			w.WriteHeader(statusCode)
			io.Copy(w, resp.Body)
		}()
	}

	// ServeHTTP ends when primary target responded
	replyWg.Wait()
}
