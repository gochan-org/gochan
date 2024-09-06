package gcsql

import (
	"net/http"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

var (
	testingPost = &Post{
		MessageRaw: "this search should match",
	}

	checkIfMatchTestCases = []filterTestCases{
		{
			name: "basic message AND check",
			filter: &Filter{
				StaffNote:   "basic message AND check",
				MatchAction: "log",
				conditions: []FilterCondition{
					{FilterID: 1, Field: "body", MatchMode: SubstrMatch, Search: "search"},
					{FilterID: 1, Field: "body", MatchMode: SubstrMatch, Search: "match"},
					{FilterID: 1, Field: "body", MatchMode: SubstrMatch, Search: "should"},
					{FilterID: 1, Field: "body", MatchMode: SubstrMatch, Search: "this"},
				},
			},
			args: checkIfMatchArgs{
				request: &http.Request{},
			},
			wantMatch: true,
		},
		{
			name: "basic message OR check",
			filter: &Filter{
				StaffNote:   "basic message OR check",
				MatchAction: "log",
				HandleIfAny: true,
				conditions: []FilterCondition{
					{FilterID: 1, Field: "body", MatchMode: SubstrMatch, Search: "aaa"},
					{FilterID: 1, Field: "body", MatchMode: SubstrMatch, Search: "bbb"},
					{FilterID: 1, Field: "body", MatchMode: SubstrMatch, Search: "search"},
					{FilterID: 1, Field: "body", MatchMode: SubstrMatch, Search: "ssss"},
				},
			},
			args: checkIfMatchArgs{
				request: &http.Request{},
			},
			wantMatch: true,
		},
	}
)

type checkIfMatchArgs struct {
	post    *Post
	upload  *Upload
	request *http.Request
}

type filterTestCases struct {
	name      string
	filter    *Filter
	args      checkIfMatchArgs
	wantMatch bool
	wantErr   bool
}

func TestFilterCheckIfMatch(t *testing.T) {
	testLog := zerolog.New(zerolog.NewTestWriter(t))
	for _, tc := range checkIfMatchTestCases {
		t.Run(tc.name, func(t *testing.T) {
			errEv := testLog.WithLevel(zerolog.ErrorLevel)
			defer errEv.Discard()
			tc.args.post = testingPost
			gotMatch, err := tc.filter.checkIfMatch(tc.args.post, tc.args.upload, tc.args.request, errEv)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				if !assert.NoError(t, err) {
					errEv.Send()
				}
			}
			assert.Equal(t, tc.wantMatch, gotMatch)
		})
	}
}
