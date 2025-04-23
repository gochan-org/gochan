package posting

import (
	"testing"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/stretchr/testify/assert"
)

type parseNameTestCase struct {
	nameAndTrip      string
	expectedName     string
	expectedTripcode string
}

func TestParseName(t *testing.T) {
	config.InitTestConfig()
	boardConfig := config.GetBoardConfig("test")
	boardConfig.ReservedTrips = map[string]string{
		"reserved": "TripOut",
	}
	config.SetBoardConfig("test", boardConfig)

	config.SetRandomSeed("lol")

	testCases := []parseNameTestCase{
		{
			nameAndTrip:      "Name#Trip",
			expectedName:     "Name",
			expectedTripcode: "piec1MorXg",
		},
		{
			nameAndTrip:      "#Trip",
			expectedName:     "",
			expectedTripcode: "piec1MorXg",
		},
		{
			nameAndTrip:  "Name",
			expectedName: "Name",
		},
		{
			nameAndTrip:  "Name#",
			expectedName: "Name",
		},
		{
			nameAndTrip:  "#",
			expectedName: "",
		},
		{
			nameAndTrip:      "Name##reserved",
			expectedName:     "Name",
			expectedTripcode: "TripOut",
		},
		{
			nameAndTrip:      "Name##notReserved",
			expectedName:     "Name",
			expectedTripcode: "MGU5NDdiYm",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.nameAndTrip, func(t *testing.T) {
			name, trip := ParseName(tC.nameAndTrip, boardConfig)
			assert.Equal(t, tC.expectedName, name)
			assert.Equal(t, tC.expectedTripcode, trip)
		})
	}
}
