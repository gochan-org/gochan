package main

import (
	"fmt"
)

type GochanVersion struct {
	Major    uint
	Minor    uint
	Revision uint
	Extra    string
}

func ParseVersion(vStr string) GochanVersion {
	var v GochanVersion
	fmt.Sscanf(vStr, "%d.%d.%d-%s", &v.Major, &v.Minor, &v.Revision, &v.Extra)
	return v
}

func (v *GochanVersion) Compare(v2 GochanVersion) int {
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

func (v *GochanVersion) CompareString(v2 string) int {
	return v.Compare(ParseVersion(v2))
}

func (v *GochanVersion) String() string {
	str := fmt.Sprintf("%d.%d", v.Major, v.Minor)
	if v.Revision > 0 {
		str += fmt.Sprintf(".%d", v.Revision)
	}
	if v.Extra != "" {
		str += "-" + v.Extra
	}
	return str
}
