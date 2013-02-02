package main

import (
	"os"
	//"syscall"
	"fmt"
)

var (
	pid, piderr uintptr
	version = 0.1
	err error
)


func main() {
	//modlogentries := []ModLogEntry
	//posts := []Post
	_,err = os.Stat("initialsetupdb.sql")
	//check if initialsetup file exists
	if err != nil {
		needs_initial_setup = false
	}
	fmt.Println("Connecting to database...(no, not really)")
	//connectToDB()
	//dbTests()
	fmt.Println("Initializing server...")
	go initServer()
	select {}
}