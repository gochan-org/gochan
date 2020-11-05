#!/usr/bin/env python

# This build script will eventually replace both the Makefile and build.ps1

import argparse
import os
from os import path
import shutil
import subprocess
import sys
import tarfile
from zipfile import ZipFile

gc_dependencies = [
	"github.com/disintegration/imaging",
	"github.com/nranchev/go-libGeoIP",
	"github.com/go-sql-driver/mysql",
	"github.com/lib/pq",
	"golang.org/x/net/html",
	"github.com/aquilax/tripcode",
	"golang.org/x/crypto/bcrypt",
	"github.com/frustra/bbcode",
	"github.com/tdewolff/minify",
	"github.com/mojocn/base64Captcha"
]

release_files = [
	"html/banned.jpg",
	"html/css",
	"html/error",
	"html/favicon2.png",
	"html/favicon.png",
	"html/firstrun.html",
	"html/hittheroad.mp3",
	"html/hittheroad.ogg",
	"html/hittheroad.wav",
	"html/javascript",
	"html/notbanned.png",
	"html/permabanned.jpg",
	"sample-configs",
	"sql",
	"templates",
	"initdb_master.sql",
	"initdb_mysql.sql",
	"initdb_postgres.sql",
	"LICENSE",
	"README.md",
]

def fs_action(action, sourcefile, destfile = ""):
	isfile = path.isfile(sourcefile) or path.islink(sourcefile)
	isdir = path.isdir(sourcefile)
	if action == "copy":
		fs_action("delete", destfile)
		if isfile:
			shutil.copy(sourcefile, destfile)
		elif isdir:
			shutil.copytree(sourcefile, destfile)
	elif action == "move":
		fs_action("delete", destfile)
		shutil.move(sourcefile, destfile)
	elif action == "mkdir":
		if isfile:
			fs_action("delete", sourcefile)
		elif not isdir:
			os.makedirs(sourcefile)
	elif action == "delete":
		if isfile:
			os.remove(sourcefile)
		elif isdir:
			shutil.rmtree(sourcefile)
	else:
		raise Exception("Invalid action, must be 'copy', 'move', 'mkdir', or 'delete'")

def run_cmd(cmd, print_output = True, realtime = False, print_command = False):
	if print_command:
		print(cmd)

	proc = subprocess.Popen(cmd, stdout = subprocess.PIPE, stderr=subprocess.STDOUT, shell = True)
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


def set_vars(goos = ""):
	""" Sets version and GOOS-related variables to be used globally"""
	global gcos
	global gcos_name # used for release, since macOS GOOS is "darwin"
	global exe
	global gochan_bin
	global gochan_exe
	global migration_bin
	global migration_exe
	global version

	if goos != "":
		os.environ["GOOS"] = goos

	gcos, gcos_status = run_cmd("go env GOOS", print_output = False)
	exe, exe_status = run_cmd("go env GOEXE", print_output = False)
	if gcos_status + exe_status != 0:
		print("Invalid GOOS value, check your GOOS environment variable")
		exit(1)
	gcos_name = gcos
	if gcos_name == "darwin":
		gcos_name = "macos"

	gochan_bin = "gochan"
	gochan_exe = "gochan" + exe
	migration_bin = "gochan-migration"
	migration_exe = "gochan-migration" + exe

	version_file = open("version", "r")
	version = version_file.read().strip()
	version_file.close()

def build(debugging = False):
	pwd = os.getcwd()
	trimpath = "-trimpath=" + pwd
	gcflags = " -gcflags=\"" + trimpath + "{}\""
	ldflags = " -ldflags=\"-X main.versionStr=" + version + "{}\""
	build_cmd = "go build -v -asmflags=" + trimpath

	if debugging:
		print("Building for", gcos,"with debugging symbols")
		gcflags = gcflags.format(" -l -N")
		ldflags = ldflags.format("")
	else:
		ldflags = ldflags.format(" -w -s")
		gcflags = gcflags.format("")
		print("Building for", gcos)
	build_cmd += gcflags + ldflags

	status = run_cmd(build_cmd + " -o " + gochan_exe + " ./cmd/gochan", realtime = True, print_command = True)[1]
	if status != 0:
		print("Failed building gochan, got status code", status)
		exit(1)

	status = run_cmd(build_cmd + " -o " + migration_exe + " ./cmd/gochan-migration", realtime = True, print_command = False)[1]
	if status != 0:
		print("Failed building gochan-migration, got status code", status)
		exit(1)
	print("Built gochan successfully")

def clean():
	print("Cleaning up")
	del_files = ["gochan", "gochan.exe", "gochan-migration", "gochan-migration.exe", "releases/", "pkg/gclog/logtest/"]
	for del_file in del_files:
		fs_action("delete", del_file)

def dependencies():
	for dep in gc_dependencies:
		run_cmd("go get -v " + dep, realtime = True, print_command = True)

def docker(option = "guestdb"):
	cmd = "docker-compose -f {} up --build"
	if option == "guestdb":
		cmd = cmd.format("docker/docker-compose-mariadb.yaml")
	elif option == "hostdb":
		cmd = cmd.format("docker/docker-compose.yml.default")
	elif option == "macos":
		cmd = cmd.format("docker/docker-compose-syncForMac.yaml")
	status = run_cmd(cmd, print_output = True, realtime = True, print_command = True)[1]
	if status != 0:
		print("Failed starting a docker container, exited with status code", status) 

def install(prefix = "/usr", document_root = "/srv/gochan"):
	if gcos == "windows" or gcos == "darwin":
		print("Installation is not currently supported for Windows and macOS, use the respective directory created by running `python build.py release`")
		exit(1)

	fs_action("mkdir", "/etc/gochan")
	fs_action("mkdir", path.join(prefix, "share/gochan"))
	fs_action("mkdir", document_root)
	fs_action("mkdir", "/var/log/gochan")
	for file in release_files:
		out_path = path.join(prefix, "share", file)
		if file.startswith("html/"):
			out_path = path.join(document_root, file.replace("html/", ""))
		
		print("Installing", file, "to", out_path)
		fs_action("copy", file, out_path)


	# print("installing sample-configs/gochan.example.json to /etc/gochan/gochan.example.json")
	# fs_action("copy")
	if(path.exists(gochan_exe) == False or path.exists(migration_exe)):
		build()
	print("Installing",gochan_exe,"to",path.join(prefix, "bin", gochan_exe))
	fs_action("copy", gochan_exe, path.join(prefix, "bin", gochan_exe))
	print("Installing",migration_exe,"to",path.join(prefix, "bin", migration_exe))
	fs_action("copy", migration_exe, path.join(prefix, "bin", migration_exe))


	print(
		"gochan successfully installed. If you haven't already, you should copy\n",
		"sample-configs/gochan.example.json to /etc/gochan/gochan.json and modify that as needed."
	)
	if gcos == "linux":
		print("If your Linux distribution has systemd, you will also need to run the following commands:\n",
			"cp sample-configs/gochan-[mysql|postgresql].service /lib/systemd/system/gochan.service\n",
			"systemctl daemon-reload\n",
			"systemctl enable gochan.service\n",
			"systemctl start gochan.service",
		)


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

def release(goos):
	set_vars(goos)
	build(False)
	release_name = gochan_bin + "-v" + version + "_" + gcos_name
	release_dir = path.join("releases", release_name)
	print("Creating release for", gcos_name)
	handle = None
	func = None
	if goos == "windows" or goos == "darwin":
		handle = ZipFile(release_dir + ".zip", "w")
		func = handle.write
	else:
		handle = tarfile.open(release_dir + ".tar.gz", "w:gz")
		func = handle.add
	func(gochan_exe, path.join(release_name, gochan_exe))
	func(migration_exe, path.join(release_name, migration_exe))
	for file in release_files:
		func(file, path.join(release_name, file))
	handle.close()

def sass(minify = False):
	sass_cmd = "sass "
	if minify:
		sass_cmd += "--style compressed "
	sass_cmd += "--no-source-map sass:html/css"
	status = run_cmd(sass_cmd, realtime = True, print_command = True)[1]
	if status != 0:
		print("Failed running sass with status", status)

def test():
	pkgs = os.listdir("pkg")
	for pkg in pkgs:
		run_cmd("go test " + path.join("./pkg", pkg), realtime = True, print_command = True)

if __name__ == "__main__":
	action = "build"
	try:
		sys.platform
		action = sys.argv.pop(1)
	except Exception: # no argument was passed
		pass
	if(action.startswith("-") == False):
		sys.argv.insert(1, action)
	set_vars()

	valid_actions = ["build", "clean", "dependencies", "docker", "install", "js", "release", "sass", "test"]
	parser = argparse.ArgumentParser(description = "gochan build script")
	parser.add_argument("action", nargs = 1, default = "build", choices = valid_actions)
	if action == "--help" or action == "-h":
		parser.print_help()
		exit(2)
	if action == "build":
		parser.add_argument("--debug", help = "build gochan and gochan-frontend with debugging symbols", action = "store_true")
		args = parser.parse_args()
		build(args.debug)
	elif action == "clean":
		clean()
		exit(0)
	elif action == "dependencies":
		dependencies()
	elif action == "docker":
		parser.add_argument("--option",
			default = "guestdb", choices = ["guestdb", "hostdb", "macos"],
			help = "create a Docker container, see docker/README.md for more info"
		)
		args = parser.parse_args()
		docker(args.option)
	elif action == "install":
		parser.add_argument("--prefix",
			default = "/usr",
			help = "install gochan to this directory and its subdirectories",
		)
		parser.add_argument("--documentroot",
			default = "/srv/gochan",
			help = "install files in ./html/ to this directory to be requested by a browser"
		)
		args = parser.parse_args()
		install(args.prefix, args.documentroot)
	elif action == "js":
		parser.add_argument("--minify", action = "store_true", help = "create a minified gochan.js")
		parser.add_argument("--watch", action = "store_true", help = "automatically rebuild when you change a file (keeps running)")
		args = parser.parse_args()
		js(args.minify, args.watch)
	elif action == "release":
		parser.add_argument("--all", help = "build releases for Windows, macOS, and Linux", action = "store_true")
		args = parser.parse_args()
		fs_action("mkdir", "releases")
		if args.all:
			release("windows")
			release("darwin")
			release("linux")
		else:
			release(gcos)
	elif action == "sass":
		parser.add_argument("--minify", action = "store_true")
		args = parser.parse_args()
		sass(args.minify)
	elif action == "test":
		test()

	args = parser.parse_args()
