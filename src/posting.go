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
	"syscall"
)

func generateTripCode(input string) string {
	input += "   " //padding
	return crypt(input,input[1:3])[3:]
}

func createThumbnail(input string, output string) bool {
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
		exitWithErrorPage("Upload file type not supported")
	}

	old_rect := image_obj.Bounds()
	defer func() {
		if _, ok := recover().(error); ok {
			//serverError()
			exitWithErrorPage("lel, internet")
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

func shortenPostForBoardPage(post *string) {

}

func makePost(w http.ResponseWriter, r *http.Request) {
	request = *r
	writer = w
	file,handler,err := request.FormFile("file")
	
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
			fmt.Println("Couldn't read file")
		} else {
			access_log.Write("Receiving post with image: "+handler.Filename+" from "+request.RemoteAddr+", referrer: "+request.Referer())
			err = ioutil.WriteFile(handler.Filename, data, 0777)
			createThumbnail(handler.Filename,"output")
			if err != nil {
				fmt.Println("Couldn't write file")
			}
		}
	}
}