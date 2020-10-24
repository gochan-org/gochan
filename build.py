#!/usr/bin/env python

# This build script will eventually replace both the Makefile and build.ps1

import argparse
import os
from os import path
import subprocess
import sys

gc_dependencies = [
	"github.com/disintegration/imaging ",
	"github.com/nranchev/go-libGeoIP ",
	"github.com/go-sql-driver/mysql ",
	"github.com/lib/pq ",
	"golang.org/x/net/html ",
	"github.com/aquilax/tripcode ",
	"golang.org/x/crypto/bcrypt ",
	"github.com/frustra/bbcode ",
	"github.com/tdewolff/minify ",
	"github.com/mojocn/base64Captcha"
]

def run_cmd(cmd, print_output = True, realtime = False, print_command = False):
	if print_command:
		print(cmd)
	use_stdout = subprocess.PIPE
	if print_output == False:
		use_stdout = open(os.devnull, 'w')

	proc = subprocess.Popen(cmd, stdout = use_stdout, stderr=subprocess.STDOUT, shell = True)
	output = ""
	status = 0
	if realtime: # print the command's output in real time, ignores print_output
		while True:
			realtime_output = proc.stdout.readline().decode("utf-8")
			if realtime_output == "" and proc.poll() is not None:
				break
			if realtime_output:
				print(realtime_output.strip())
				output += realtime_output
			status = proc.poll()
	else: # wait until the command is finished to print the output
		output = proc.communicate()[0]
		if output is not None:
			output = output.decode("utf-8").strip()
		else:
			output = ""
		status = proc.wait()
		if output != "" and print_output:
			print(output)
	if status is None:
		status = 0
	return (output, status)

def build(debugging = False):
	use_cmd = build_cmd
	if debugging:
		use_cmd = dbg_build_cmd
		print("Building for", gcos,"with debugging symbols")
	else:
		print("Building for", gcos)
	
	status = run_cmd(use_cmd + " -o " + gochan_exe + " ./cmd/gochan", realtime = True, print_command = True)[1]
	if status != 0:
		print("Failed building gochan, got status code", status)
		exit(1)

	status = run_cmd(use_cmd + " -o " + migration_exe + " ./cmd/gochan-migration", realtime = True, print_command = True)[1]
	if status != 0:
		print("Failed building gochan-migration, got status code", status)
		exit(1)
	print("Built gochan successfully")

def clean():
	print("Cleaning up")
	del_files = ["gochan", "gochan.exe", "gochan-migration", "gochan-migration.exe"]
	for del_file in del_files:
		if path.exists(del_file):
			os.remove(del_file)

def dependencies():
	for dep in gc_dependencies:
		run_cmd("go get -v " + dep, realtime = True)

def docker(option = "guestdb"):
	cmd = ""
	if option == "guestdb":
		cmd = "docker-compose -f docker/docker-compose-mariadb.yaml up --build"
	elif option == "hostdb":
		cmd = "docker-compose -f docker/docker-compose.yml.default up --build"
	elif option == "macos":
		cmd = "docker-compose -f docker/docker-compose-syncForMac.yaml up --build"
	status = run_cmd(cmd, print_output = True, realtime = True, print_command = True)[1]
	if status != 0:
		print("Failed starting a docker container, exited with status code", status) 

def install():
	pass

def js(minify = False, watch = False):
	print("Transpiling JS")
	npm_cmd = "npm --prefix frontend/ run build"
	if minify:
		npm_cmd += "-minify"
	if watch:
		npm_cmd += "-watch"
	status = run_cmd(npm_cmd, True, True, True)[1]
	if status != 0:
		print("JS transpiling failed with status", status)

def release(all = True):
	global gcos
	print("Creating releases for GOOS", gcos)

def sass(minify = False):
	sass_cmd = "sass "
	if minify:
		sass_cmd += "--style compressed "
	status = run_cmd(sass_cmd + "--no-source-map sass:html/css", realtime = True, print_command = True)[1]
	if status != 0:
		print("Failed running sass with status", status)

def test():
	run_cmd("go test ./pkg/gcutil/", realtime = True, print_command = True)

if __name__ == "__main__":
	global gcos
	global gcos_name # used for release, since macOS GOOS is "darwin"
	global exe
	global gochan_bin
	global gochan_exe
	global migration_bin
	global migration_exe
	global build_cmd
	global dbg_build_cmd

	action = "build"
	try:
		action = sys.argv.pop(1)
	except Exception: # no argument was passed
		pass

	if(action.startswith("-") == False):
		sys.argv.insert(1, action)

	valid_actions = ["build", "clean", "dependencies", "docker", "install", "js", "release", "sass", "test"]
	parser = argparse.ArgumentParser(description = "gochan build script")
	parser.add_argument("action",
		nargs = 1,
		default = "build",
		choices = valid_actions
	)

	if action == "--help" or action == "-h":
		parser.print_help()
		exit(2)
	
	gcos, gcos_status = run_cmd("go env GOOS")
	exe, exe_status = run_cmd("go env GOEXE")
	if gcos_status + exe_status != 0:
		print("Invalid GOOS value, check your GOOS environment variable")
		exit(1)

	gochan_bin = "gochan"
	gochan_exe = "gochan" + exe
	migration_exe = "gochan-migration" + exe

	version_file = open("version", "r")
	version = version_file.read().strip()
	version_file.close()

	pwd = os.getcwd()
	trimpath = "-trimpath=" + pwd
	ldflags = "-X main.versionStr=" + version
	build_prefix = "go build -v -asmflags=" + trimpath + " -gcflags="

	build_cmd = build_prefix + trimpath + " -ldflags=\"" + ldflags + " -w -s\""
	dbg_build_cmd = build_prefix + "\"" + trimpath + " -l -N\" -ldflags=\"" + ldflags + "\""

	if action == "build":
		parser.add_argument("--debug", type = bool, default = False, nargs = "?")
		args = parser.parse_args()
		build(args.debug is not False)
	elif action == "clean":
		clean()
		exit(0)
	elif action == "dependencies":
		dependencies()
	elif action == "docker":
		parser.add_argument("--option", type = str, default = "guestdb", choices = ["guestdb", "hostdb", "macos"])
		args = parser.parse_args()
		docker(args.option)
	elif action == "install":
		install()
	elif action == "js":
		parser.add_argument("--minify", type = bool, default = False, nargs = "?")
		parser.add_argument("--watch", type = bool, default = False, nargs = "?")
		args = parser.parse_args()
		js(args.minify is not False, args.watch is not False)
	elif action == "release":
		parser.add_argument("--all", type = bool, default = False, nargs = "?")
		args = parser.parse_args()
		release(args.all is not False)
	elif action == "sass":
		parser.add_argument("--minify", type = bool, default = False, nargs = "?")
		args = parser.parse_args()
		sass(args.minify is not False)
	elif action == "test":
		test()

	args = parser.parse_args()
