package building

type catalogThreadData struct {
	Replies       int `json:"replies"`
	Images        int `json:"images"`
	OmittedPosts  int `json:"omitted_posts"`
	OmittedImages int `json:"omitted_images"`
	Sticky        int `json:"sticky"`
	Locked        int `json:"locked"`
	numPages      int
}

type catalogPage struct {
	PageNum int                 `json:"page"`
	Threads []catalogThreadData `json:"threads"`
}

type boardCatalog []catalogPage
