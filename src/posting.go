package main

import (
	"net/http"
	"io/ioutil"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/gif"
	"image/png"
	"math/rand"
	"os"
	"path"
	"./lib/resize"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	UnsupportedFiletypeError =  errors.New("Upload filetype not supported")
	FileWriteError = errors.New("Couldn't write file.")
)

func generateTripCode(input string) string {
	input += "   " //padding
	return crypt(input,input[1:3])[3:]
}

func buildBoardPages(boardid int) {
	
}

func buildThread(op_post PostTable) (err error) {
	threadid_str := strconv.Itoa(op_post.ID)
	thread_posts := getPostArr("`deleted_timestamp` IS NULL AND (`parentid` = "+threadid_str+" OR `id` = "+threadid_str+") AND `boardid` = "+strconv.Itoa(op_post.BoardID))
	board_arr := getBoardArr("")
	sections_arr := getSectionArr("")

	op_id := strconv.Itoa(op_post.ID)
	var board_dir string
	for _,board_i := range board_arr {
		board := board_i.(BoardsTable)
		if board.ID == op_post.BoardID {
			board_dir = board.Dir
			break
		}
	}

    var interfaces []interface{}
    interfaces = append(interfaces, config)
    interfaces = append(interfaces, thread_posts)
    interfaces = append(interfaces, &Wrapper{IName:"boards", Data: board_arr})
    interfaces = append(interfaces, &Wrapper{IName:"sections", Data: sections_arr})

	wrapped := &Wrapper{IName: "threadpage",Data: interfaces}
	os.Remove("html/"+board_dir+"/res/"+op_id+".html")
	
	thread_file,err := os.OpenFile("html/"+board_dir+"/res/"+op_id+".html",os.O_CREATE|os.O_RDWR,0777)
	err = img_thread_tmpl.Execute(thread_file,wrapped)
	if err == nil {
		if err != nil {
			return err
		} else {
			return nil
		}
	}
	return
}

// checks to see if the poster's tripcode/name is banned, if the IP is banned, or if the file checksum is banned
func checkBannedStatus(post PostTable) bool {
	return false
}

type ThumbnailPre struct {
	Filename_old string
	Filename_new string
	Filepath string
	Width int
	Height int
	Obj image.Image
	ThumbObj image.Image
}

func loadImage(file *os.File) (image.Image,error) {
	filetype := file.Name()[len(file.Name())-3:len(file.Name())]
	var image_obj image.Image
	var err error

	if filetype == "gif" {
		image_obj,err = gif.Decode(file)
	} else if filetype == "jpeg" || filetype == "jpg" {
		image_obj,err = jpeg.Decode(file)
	} else if filetype == "png" {
		image_obj,err = png.Decode(file)
	} else {
		image_obj = nil
		err = UnsupportedFiletypeError
	}
	return image_obj,err
}

func saveImage(path string, image_obj *image.Image) error {
	outwriter,err := os.OpenFile(path, os.O_RDWR|os.O_CREATE,0777)
	if err == nil {
		filetype := path[len(path)-4:len(path)]
		if filetype == ".gif" {
			//because Go doesn't come with a GIF writer :c
			jpeg.Encode(outwriter, *image_obj, &jpeg.Options{Quality: 80})
		} else if filetype == ".jpg" || filetype == "jpeg" {
			jpeg.Encode(outwriter, *image_obj, &jpeg.Options{Quality: 80})
		} else if filetype == ".png" {
			png.Encode(outwriter, *image_obj)
		} else {
			return UnsupportedFiletypeError
		}
	}
	return err
}

func createThumbnail(image_obj image.Image, size string) image.Image {
	var thumb_width int
	var thumb_height int

	switch {
		case size == "op":
			thumb_width = config.ThumbWidth
			thumb_height = config.ThumbHeight
		case size == "reply":
			thumb_width = config.ThumbWidth_reply
			thumb_height = config.ThumbHeight_reply
		case size == "catalog":
			thumb_width = config.ThumbWidth_catalog
			thumb_height = config.ThumbHeight_catalog
	}
	old_rect := image_obj.Bounds()
	if thumb_width >= old_rect.Max.X && thumb_height >= old_rect.Max.Y {
		return image_obj
	}
	
	thumb_w,thumb_h := getThumbnailSize(old_rect.Max.X,old_rect.Max.Y)
	image_obj = resize.Resize(image_obj, image.Rect(0,0,old_rect.Max.X,old_rect.Max.Y), thumb_w,thumb_h)
	return image_obj
}


func getFiletype(name string) string {
	filetype := strings.ToLower(name[len(name)-4:len(name)])
	if filetype == ".gif" {
		return "gif"
	} else if filetype == ".jpg" || filetype == "jpeg" {
		return "jpg"
	} else if filetype == ".png" {
		return "png"
	} else {
		return name[len(name)-3:len(name)]
	}
}

func getNewFilename() string {
	now := time.Now().Unix()
	rand.Seed(now)
	return strconv.Itoa(int(now))+strconv.Itoa(int(rand.Intn(98)+1))
}

// find out what out thumbnail's width and height should be, partially ripped from Kusaba X
func getThumbnailSize(w int, h int) (new_w int, new_h int) {
	if w == h {
		new_w = config.ThumbWidth
		new_h = config.ThumbWidth
	} else {
		var percent float32
		if (w > h) {
			percent = float32(config.ThumbWidth) / float32(w)
		} else {
			percent = float32(config.ThumbWidth) / float32(h)
		}
		new_w = int(float32(w) * percent)
		new_h = int(float32(h) * percent)
	}
	return
}

// inserts prepared post object into the SQL table so that it can be rendered
func insertPost(post PostTable) {
	post = sanitizePost(post)
	post_sql_str := "INSERT INTO `"+config.DBprefix+"posts` (`boardid`,`parentid`,`name`,`tripcode`,`email`,`subject`,`message`,`password`"
	if post.Filename != "" {
		post_sql_str += ",`filename`,`filename_original`,`file_checksum`,`filesize`,`image_w`,`image_h`,`thumb_w`,`thumb_h`"
	}
	post_sql_str += ",`ip`"
	post_sql_str += ",`timestamp`,`poster_authority`,`stickied`,`locked`) VALUES("+strconv.Itoa(post.BoardID)+","+strconv.Itoa(post.ParentID)+",'"+post.Name+"','"+post.Tripcode+"','"+post.Email+"','"+post.Subject+"','"+post.Message+"','"+post.Password+"'"
	if post.Filename != "" {
		post_sql_str += ",'"+post.Filename+"','"+post.FilenameOriginal+"','"+post.FileChecksum+"',"+strconv.Itoa(int(post.Filesize))+","+strconv.Itoa(post.ImageW)+","+strconv.Itoa(post.ImageH)+","+strconv.Itoa(post.ThumbW)+","+strconv.Itoa(post.ThumbH)
	}
	post_sql_str += ",'"+post.IP+"','"+post.Timestamp+"',"+strconv.Itoa(post.PosterAuthority)+","
	if post.Stickied {
		post_sql_str += "1,"
	} else {
		post_sql_str += "0,"
	}
	if post.Locked {
		post_sql_str += "1);"
	} else {
		post_sql_str += "0);"
	}
	fmt.Println(post_sql_str)
	//_,err := db.Start(post_sql_str)
}

// calls db.Escape() on relevant post members to prevent SQL injection
func sanitizePost(post PostTable) PostTable {
	return post
}

func shortenPostForBoardPage(post *string) {

}

func makePost(w http.ResponseWriter, r *http.Request) {
	request = *r
	writer = w
	request.ParseForm()
	var post PostTable
	post.IName = "post"
	post.ParentID,_ = strconv.Atoi(request.FormValue("threadid"))
	post.BoardID,_ = strconv.Atoi(request.FormValue("boardid"))
	post.Name = db.Escape(request.FormValue("postname"))
	post.Email = db.Escape(request.FormValue("postemail"))
	post.Subject = db.Escape(request.FormValue("postsubject"))
	post.Message = db.Escape(request.FormValue("postmsg"))
	// TODO: change this to a checksum
	post.Password = db.Escape(request.FormValue("postpassword"))
	post.IP = request.RemoteAddr
	post.Timestamp = getSQLDateTime()
	post.PosterAuthority = getStaffRank()
	post.Bumped = post.Timestamp
	post.Stickied = request.FormValue("modstickied") == "on"
	post.Locked = request.FormValue("modlocked") == "on"

	//post has no referrer, or has a referrer from a different domain, probably a spambot
	if request.Referer() == "" || request.Referer()[7:len(config.Domain)+7] != config.Domain {
		access_log.Write("Rejected post from possible spambot @ : "+request.RemoteAddr)
		//TODO: insert post into temporary post table and add to report list
	}
	file,handler,uploaderr := request.FormFile("imagefile")
	if uploaderr != nil {
		// no file was uploaded
		fmt.Println(uploaderr.Error())
		post.Filename = ""
		access_log.Write("Receiving post from "+request.RemoteAddr+", referred from: "+request.Referer())
	} else {
		data,err := ioutil.ReadAll(file)
		if err != nil {
			exitWithErrorPage(w,"Couldn't read file")
		} else {
			post.FilenameOriginal = handler.Filename
			filetype := post.FilenameOriginal[len(post.FilenameOriginal)-3:len(post.FilenameOriginal)]
			
			post.Filename = getNewFilename()+"."+getFiletype(post.FilenameOriginal)
			
			file_path := path.Join(config.DocumentRoot,"/"+getBoardArr("`id` = "+request.FormValue("boardid"))[0].(BoardsTable).Dir+"/src/",post.Filename)
			thumb_path := path.Join(config.DocumentRoot,"/"+getBoardArr("`id` = "+request.FormValue("boardid"))[0].(BoardsTable).Dir+"/thumb/",strings.Replace(post.Filename,"."+filetype,"t."+filetype,-1))

			err := ioutil.WriteFile(file_path, data, 0777)
			if err != nil {
				exitWithErrorPage(w,"Couldn't write file.")
			}

			image_file,err := os.OpenFile(file_path, os.O_RDONLY, 0)
			if err != nil {
				exitWithErrorPage(w,"Couldn't read saved file")
			}
			
			img,err := loadImage(image_file)
			if err != nil {
				exitWithErrorPage(w,err.Error())
			} else {
				//post.FileChecksum string
				stat,err := image_file.Stat()
				if err != nil {
					exitWithErrorPage(w,err.Error())
				} else {
					post.Filesize = int(stat.Size())
				}
				post.ThumbW,post.ThumbH = getThumbnailSize(post.ImageW,post.ImageH)

				access_log.Write("Receiving post with image: "+handler.Filename+" from "+request.RemoteAddr+", referrer: "+request.Referer())
				

				if config.ThumbWidth >= img.Bounds().Max.X && config.ThumbHeight >= img.Bounds().Max.Y {
					err := syscall.Symlink(file_path,thumb_path)
					if err != nil {
						exitWithErrorPage(w,err.Error())
					}
				} else {
					thumbnail := createThumbnail(img,"op")
					err = saveImage(thumb_path, &thumbnail)
					if err != nil {
						exitWithErrorPage(w,err.Error())
					} else {
						http.Redirect(writer,&request,"/test/res/1.html",http.StatusFound)
					}
				}
			}
		}
	}
}