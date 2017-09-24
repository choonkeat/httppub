package broadcast

import "net/http"

type header interface {
	Header() http.Header
}

func mergeHeaders(target http.Header, source http.Header) http.Header {
	for k, varray := range source {
		for _, v := range varray {
			target.Set(k, v)
		}
	}
	return target
}
