// used for version parsing, printing, and comparison

package main

import (
	"fmt"
)

type GochanVersion struct {
	Major    int
	Minor    int
	Revision int
	Extra    string
}

func ParseVersion(vStr string) GochanVersion {
	var v GochanVersion
	fmt.Sscanf(vStr, "%d.%d.%d-%s", &v.Major, &v.Minor, &v.Revision, &v.Extra)
	v.Normalize()
	return v
}

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
	return valid
}

func (v *GochanVersion) Compare(v2 GochanVersion) int {
	v.Normalize()
	v2.Normalize()
	if v.Major > v2.Major {
		return 1
	}
	if v.Major < v2.Major {
		return -1
	}
	if v.Minor > v2.Minor {
		return 1
	}
	if v.Minor < v2.Minor {
		return -1
	}
	if v.Revision > v2.Revision {
		return 1
	}
	if v.Revision < v2.Revision {
		return -1
	}
	return 0
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
