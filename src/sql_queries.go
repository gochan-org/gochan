package main

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	libgeo "github.com/nranchev/go-libGeoIP"
	"golang.org/x/crypto/bcrypt"
)

func GetTopPosts(boardId int, sortByDescending bool) (posts []Post, err error) {
	//TODO sort by bump
}

func GetExistingReplies(top_post int) (posts []Post, err error) {
	//TODO
}

func GetExistingRepliesLimitedRev(top_post int, limit int) (posts []Post, err error) {
	//TODO
}

func GetSpecificTopPost(id int) (posts []Post, err error) {
	//TODO
}

func GetSpecificPost(id int) (posts []Post, err error) {
	//TODO
}

func GetAllNondeletedMessageRaw() (messages []MessagePostContainer, err error) {
	//TODO
}

func SetMessages(messages []MessagePostContainer) (err error) {
	//TODO
}
