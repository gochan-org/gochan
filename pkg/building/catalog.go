package building

import "github.com/gochan-org/gochan/pkg/gcsql"

type catalogThreadData struct {
	Replies       int `json:"replies"`
	Images        int `json:"images"`
	OmittedPosts  int `json:"omitted_posts"`  // posts in the thread but not shown on the board page
	OmittedImages int `json:"omitted_images"` // uploads in the thread but not shown on the board page
	Sticky        int `json:"sticky"`
	Locked        int `json:"locked"`
	// numPages      int
	posts   []gcsql.Post
	uploads []gcsql.Upload
}

type catalogPage struct {
	PageNum int                 `json:"page"`
	Threads []catalogThreadData `json:"threads"`
}

type boardCatalog struct {
	pages       []catalogPage // this array gets marshalled, not the boardCatalog object
	numPages    int
	currentPage int
}

// fillPages fills the catalog's pages array with pages of the specified size, with the remainder
// on the last page
func (catalog *boardCatalog) fillPages(threadsPerPage int, threads []catalogThreadData) {
	catalog.pages = []catalogPage{} // clear the array if it isn't already
	catalog.numPages = len(threads) / threadsPerPage
	remainder := len(threads) % threadsPerPage
	currentThreadIndex := 0
	var i int
	for i = 0; i < catalog.numPages; i++ {
		catalog.pages = append(catalog.pages,
			catalogPage{
				PageNum: i + 1,
				Threads: threads[currentThreadIndex : currentThreadIndex+threadsPerPage],
			},
		)
		currentThreadIndex += threadsPerPage
	}
	if remainder > 0 {
		catalog.pages = append(catalog.pages,
			catalogPage{
				PageNum: i + 1,
				Threads: threads[len(threads)-remainder:],
			},
		)
	}
}

// func paginateBoards(threadsPerPage int, threads []catalogThreadData) [][]catalogThreadData {
// 	var paginatedThreads [][]catalogThreadData
// 	numArrays := len(threads) / threadsPerPage
// 	remainder := len(threads) % threadsPerPage
// 	currentIndex := 0
// 	for l := 0; l < numArrays; l++ {
// 		paginatedThreads = append(paginatedThreads,
// 			threads[currentIndex:currentIndex+threadsPerPage])
// 		currentIndex += threadsPerPage
// 	}
// 	if remainder > 0 {
// 		paginatedThreads = append(paginatedThreads, threads[len(threads)-remainder:])
// 	}
// 	return paginatedThreads
// }
