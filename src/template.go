package main

func getStyleLinks(stylesheet string) (links_str string) {
	num_styles := len(styles_arr)
	for l := 0; l < num_styles; l++ {
		links_str += "<link rel=\""
		if l > 0 {
			links_str += "alternate "
		}
		links_str += "stylesheet\" href=\"/css/"+styles_arr[l]+"/"+stylesheet+".css\" />\n"
	}
	return
}