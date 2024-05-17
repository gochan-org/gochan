// used for version parsing, printing, and comparison

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type versionTestCase struct {
	versionStr     string
	expectMajor    int
	expectMinor    int
	expectRevision int
	expectExtra    string
	expectString   string
	expectInvalid  bool
}

func (tC *versionTestCase) run(t *testing.T) {
	version := ParseVersion(tC.versionStr)
	valid := version.Normalize()
	assert.NotEqual(t, tC.expectInvalid, valid)
	if tC.expectInvalid {
		return
	}
	assert.Equal(t, tC.expectMajor, version.Major)
	assert.Equal(t, tC.expectMinor, version.Minor)
	assert.Equal(t, tC.expectRevision, version.Revision)
	assert.Equal(t, tC.expectExtra, version.Extra)
	assert.Equal(t, tC.expectString, version.String())
}

func TestParseVersion(t *testing.T) {
	tests := []versionTestCase{
		{
			versionStr:     "1.2.3-extra",
			expectMajor:    1,
			expectMinor:    2,
			expectRevision: 3,
			expectExtra:    "extra",
			expectString:   "1.2.3-extra",
		},
		{
			versionStr:     "1.2.3",
			expectMajor:    1,
			expectMinor:    2,
			expectRevision: 3,
			expectString:   "1.2.3",
		},
		{
			versionStr:   "1.2",
			expectMajor:  1,
			expectMinor:  2,
			expectString: "1.2",
		},
		{
			versionStr:   "1",
			expectMajor:  1,
			expectString: "1.0",
		},
		{
			versionStr:    "-1.-1.-1",
			expectInvalid: true,
		},
	}
	for _, tC := range tests {
		t.Run(tC.versionStr, tC.run)
	}
}

type versionCompareTestCase struct {
	desc      string
	v1Str     string
	v2Str     string
	expectCmp int
}

func (tC *versionCompareTestCase) run(t *testing.T) {
	v1 := ParseVersion(tC.v1Str)
	cmp := v1.CompareString(tC.v2Str)
	assert.Equal(t, tC.expectCmp, cmp)
}

func TestVersionCompare(t *testing.T) {
	tests := []versionCompareTestCase{
		{
			desc:      "v1 > v2 major",
			v1Str:     "3.0",
			v2Str:     "2.0",
			expectCmp: 1,
		},
		{
			desc:      "v1 > v2 minor",
			v1Str:     "3.1",
			v2Str:     "3.0",
			expectCmp: 1,
		},
		{
			desc:      "v1 > v2 revision",
			v1Str:     "3.1.1",
			v2Str:     "3.1",
			expectCmp: 1,
		},
		{
			desc:      "v1 < v2 major",
			v1Str:     "2.0",
			v2Str:     "3.0",
			expectCmp: -1,
		},
		{
			desc:      "v1 < v2 minor",
			v1Str:     "3.0",
			v2Str:     "3.1",
			expectCmp: -1,
		},
		{
			desc:      "v1 < v2 revision",
			v1Str:     "3.1",
			v2Str:     "3.1.1",
			expectCmp: -1,
		},
		{
			desc:      "v1 = v2",
			v1Str:     "3.1.1",
			v2Str:     "3.1.1",
			expectCmp: 0,
		},
	}
	for _, tC := range tests {
		t.Run(tC.desc, tC.run)
	}
}
