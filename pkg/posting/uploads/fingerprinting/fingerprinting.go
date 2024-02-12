package fingerprinting

import (
	"fmt"
	"image"
	"net/http"

	"github.com/gochan-org/gochan/pkg/gcsql"
)

const (
	defaultHashLength = 16
)

var (
	fingerprinterHandlers map[string]UploadFingerprinter
)

type UploadFingerprinter interface {
	// Init initializes the fingerprinter, and accepts options assumed to be in gochan.json
	Init(options map[string]any) error
	// IsCompatible returns true if the fingerprinter is able to handle the incoming upload
	IsCompatible(upload *gcsql.Upload) bool
	// CheckFile scans the incoming file and scans it against files in DBPREFIXfile_ban
	// with the fingerprinter's id. It returns true if the file is banned
	CheckFile(source *FingerprintSource, board string) (*gcsql.FileBan, error)
	// Close closes the fingerprinter. This may or may not be necessary
	Close() error
}

type FingerprintSource struct {
	FilePath string
	Img      image.Image
	Request  *http.Request
}

// RegisterFingerprinter registers the given id. It must be in the configuration,
// in the FingerprinterOptions map
func RegisterFingerprinter(id string, handler UploadFingerprinter) error {
	_, ok := fingerprinterHandlers[id]
	if ok {
		return fmt.Errorf("a fingerprinter has already been registered to the ID %q", id)
	}
	fingerprinterHandlers[id] = handler
	return nil
}

func init() {
	RegisterFingerprinter("ahash", &ahashHandler{})
}
