// used for version parsing, printing, and comparison

package config

import (
	"fmt"
)

const (
	VersionEqual = 0
	VersionNewer = 1
	VersionOlder = -1
)

type GochanVersion struct {
	Major    int
	Minor    int
	Revision int
	Extra    string
}

func ParseVersion(vStr string) *GochanVersion {
	var v GochanVersion
	fmt.Sscanf(vStr, "%d.%d.%d-%s", &v.Major, &v.Minor, &v.Revision, &v.Extra)
	v.Normalize()
	return &v
}

// Normalize checks to make sure that the version is legitimate, i.e. fields > 0
func (v *GochanVersion) Normalize() bool {
	valid := true
	if v.Major < 0 {
		v.Major = 0
		valid = false
	}
	if v.Minor < 0 {
		v.Minor = 0
		valid = false
	}
	if v.Revision < 0 {
		v.Revision = 0
	}
	if v.Revision > 0 && v.Minor == 0 && v.Major == 0 {
		v.Minor = 1
		valid = false
	}
	return valid
}

// Compare compares v to v2 and returns 1 if it is newer, -1 if it older, and 0
// if they are equal
func (v *GochanVersion) Compare(v2 *GochanVersion) int {
	v.Normalize()
	v2.Normalize()
	if v.Major > v2.Major {
		return VersionNewer
	}
	if v.Major < v2.Major {
		return VersionOlder
	}
	if v.Minor > v2.Minor {
		return VersionNewer
	}
	if v.Minor < v2.Minor {
		return VersionOlder
	}
	if v.Revision > v2.Revision {
		return VersionNewer
	}
	if v.Revision < v2.Revision {
		return VersionOlder
	}
	return VersionEqual
}

func (v *GochanVersion) CompareString(v2str string) int {
	v.Normalize()
	v2 := ParseVersion(v2str)
	v2.Normalize()
	return v.Compare(v2)
}

func (v *GochanVersion) String() string {
	v.Normalize()
	str := fmt.Sprintf("%d.%d", v.Major, v.Minor)
	if v.Revision > 0 {
		str += fmt.Sprintf(".%d", v.Revision)
	}
	if v.Extra != "" {
		str += "-" + v.Extra
	}
	return str
}
