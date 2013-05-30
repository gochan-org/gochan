package main

import (
	"net/http"
	"io/ioutil"
	"fmt"
	"image"
	"image/jpeg"
	"image/gif"
	"image/png"
	"os"
	"./lib/resize"
	"strconv"
	"syscall"
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


func createThumbnail(w http.ResponseWriter, input string, output string) bool {
	var image_obj image.Image
	failed := false
	handle,err := os.Open(input)

	if err != nil {
		return false
	}
	defer func() {
		if _, ok := recover().(error); ok {		
			handle.Close()
			failed = true
		}
	}()
	if failed {
		error_log.Write("Failed to create thumbnail")
		return false
	}

	filetype := input[len(input)-3:len(input)]
	if filetype == "gif" {
		image_obj,_ = gif.Decode(handle)
	} else if filetype == "jpeg" || filetype == "jpg" {
		image_obj,_ = jpeg.Decode(handle)
	} else if filetype == "png" {
		image_obj,_ = png.Decode(handle)
	} else {
		exitWithErrorPage(w, "Upload file type not supported")
	}

	old_rect := image_obj.Bounds()
	defer func() {
		if _, ok := recover().(error); ok {
			//serverError()
			exitWithErrorPage(w, "lel, internet")
		}
	}()
	if config.ThumbWidth >= old_rect.Max.X && config.ThumbHeight >= old_rect.Max.Y {
		err := syscall.Symlink(input,output)
		if err != nil {
			error_log.Write(fmt.Sprintf("Error, couldn't create symlink to %s, %s", input, err.Error()))
			return false
		} else {
			return true
		}

	}
	
	thumb_w,thumb_h := getThumbnailSize(old_rect.Max.X,old_rect.Max.Y)
	image_obj = resize.Resize(image_obj, image.Rect(0,0,old_rect.Max.X,old_rect.Max.Y), thumb_w,thumb_h)
	
	outwriter,_ := os.OpenFile(output, os.O_RDWR|os.O_CREATE,0777)
	if filetype == "gif" {
		//because Go doesn't come with a GIF writer :c
		jpeg.Encode(outwriter, image_obj, &jpeg.Options{Quality: 80})
	} else if filetype == "jpg" || filetype == "jpeg" {
		jpeg.Encode(outwriter,image_obj, &jpeg.Options{Quality: 80})
	} else if filetype == "png" {
		png.Encode(outwriter,image_obj)
	}
	return false
}

//find out what out thumbnail's width and height should be, partially ripped from Kusaba X

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
		//fmt.Printf("Old width: %d\nOld height: %d\nPercent: %f\nWidth: %d\nHeight: %d\n",w,h,percent*100,new_w,new_h)		
	}
	return
}

func insertPost(post PostTable) {

}

func shortenPostForBoardPage(post *string) {

}

func makePost(w http.ResponseWriter, r *http.Request) {
	request = *r
	writer = w
	file,handler,err := request.FormFile("file")
	/*threadid := db.Escape(request.FormValue("threadid"))
	boardid := db.Escape(request.FormValue("boardid"))
	postname := db.Escape(request.FormValue("postname"))
	postemail := db.Escape(request.FormValue("postemail"))
	postmsg := db.Escape(request.FormValue("postmsg"))
	imagefile := db.Escape(request.FormValue("imagefile"))*/

	//post has no referrer, or has a referrer from a different domain, probably a spambot
	if request.Referer() == "" || request.Referer()[7:len(config.Domain)+7] != config.Domain {
		access_log.Write("Rejected post from possible spambot @ : "+request.RemoteAddr)
		//TODO: insert post into temporary post table and add to report list
	}

	//no file was uploaded
	if err != nil {
		access_log.Write("Receiving post from "+request.RemoteAddr+", referred from: "+request.Referer())
	} else {
		data,err := ioutil.ReadAll(file)
		if err != nil {
			exitWithErrorPage(w,"Couldn't read file")
		} else {
			access_log.Write("Receiving post with image: "+handler.Filename+" from "+request.RemoteAddr+", referrer: "+request.Referer())
			err = ioutil.WriteFile(handler.Filename, data, 0777)
			createThumbnail(w, handler.Filename,"output")
			if err != nil {
				exitWithErrorPage(w,"Couldn't write file")
			}
		}
	}
}