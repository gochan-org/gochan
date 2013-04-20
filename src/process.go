package main

// #define _GNU_SOURCE
// #include <stdio.h>
// #include <unistd.h>
// #include <sys/types.h>
import "C"

var pid uintptr
/*
func fork() int {
	return C.GoInt(C.fork())
}

func setsid() {
	C.setsid()
}*/