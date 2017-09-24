package broadcast

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeHeaders(t *testing.T) {
	testCases := []struct {
		givenTarget, givenSource http.Header
		expectedResult           http.Header
	}{
		{
			givenTarget: http.Header{
				"X-Greeting": []string{"hello", "world"},
			},
			givenSource: http.Header{
				"key-string": []string{"hi"},
				"X-Greeting": []string{"aloha"},
			},
			expectedResult: http.Header{
				"X-Greeting": []string{"aloha"}, // overwrite
				"Key-String": []string{"hi"},    // note capitalize!
			},
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			mergeHeaders(tc.givenTarget, tc.givenSource)
			assert.Equal(t, tc.expectedResult, tc.givenTarget)
		})
	}
}
