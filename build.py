#!/usr/bin/env python3

"""
Gochan build/install/maintenance script
For a list of commands, run 
    python3 build.py --help
For information on a specific command, run
    python3 build.py <command> --help
See README.md for more info
"""

import argparse
import errno
import os
from os import path
from pathlib import Path
import shutil
import subprocess
import sys
import traceback

release_files = (
	"html/css/",
	"html/error/",
	"html/static/",
	"html/favicon2.png",
	"html/favicon.png",
	"html/firstrun.html",
	"html/js/",
	"examples/configs/",
	"examples/plugins/",
	"templates/",
	"sql/initdb_mysql.sql",
	"sql/initdb_postgres.sql",
	"sql/initdb_sqlite3.sql",
	"sql/reset_views.sql",
	"sql/README.md",
	"LICENSE",
	"README.md",
)

GOCHAN_VERSION = "4.0.2"
DATABASE_VERSION = "4"  # stored in DBNAME.DBPREFIXdatabase_version

PATH_NOTHING = -1
PATH_UNKNOWN = 0
PATH_FILE = 1
PATH_DIR = 2
PATH_LINK = 4


gcos = ""
gcos_name = ""  # used for release, since macOS GOOS is "darwin"
exe = ""
gochan_bin = ""
gochan_exe = ""
migration_bin = ""
migration_exe = ""


def pathinfo(loc):
	i = PATH_UNKNOWN
	if not path.exists(loc):
		return PATH_NOTHING
	if path.islink(loc):
		i |= PATH_LINK
	if path.isfile(loc):
		i |= PATH_FILE
	elif path.isdir(loc):
		i |= PATH_DIR
	else:
		i = PATH_UNKNOWN
	return i


def delete(delpath):
	"""
	Deletes the given file, link, or directory and silently fails if nothing exists.
	Returns the path info as well
	"""
	pinfo = pathinfo(delpath)
	if pinfo == PATH_NOTHING:
		return PATH_NOTHING
	if pinfo & PATH_FILE > 0 or pinfo & PATH_LINK > 0:
		os.remove(delpath)
		return pinfo
	if pinfo & PATH_DIR > 0:
		shutil.rmtree(delpath)
		return pinfo
	return PATH_UNKNOWN


def mkdir(dirpath, force=False):
	if path.exists(dirpath):
		if force:
			delete(dirpath)
		else:
			return
	os.makedirs(dirpath)


def copy(source, dest):
	"""
	Copy source to dest, overwriting dest if source and dest are files, and merging
	them if source is a directory and dest is a directory that already exists, overwriting
	any conflicting files
	"""
	srcinfo = pathinfo(source)
	destinfo = pathinfo(dest)
	if srcinfo == PATH_NOTHING:
		raise FileNotFoundError(errno.ENOENT, os.strerror(errno.ENOENT), source)
	if srcinfo & PATH_FILE > 0 or srcinfo & PATH_LINK > 0:
		shutil.copy(source, dest)
		return
	if srcinfo & PATH_DIR > 0:
		if destinfo == PATH_NOTHING:
			mkdir(dest)
		else:
			for root, dirs, files in os.walk(source):
				mkdir(path.join(dest, root))
				for dirpath in dirs:
					mkdir(path.join(dest, root, dirpath))
				for file in files:
					shutil.copy(path.join(root, file), path.join(dest, root, file))


def symlink(target, link):
	"""Create symbolic link at `link` that points to `target`"""
	targetinfo = pathinfo(target)
	linkinfo = pathinfo(link)
	if target == PATH_NOTHING:
		raise FileNotFoundError(errno.ENOENT, os.strerror(errno.ENOENT), target)
	if linkinfo != PATH_NOTHING:
		delete(link)
	elif linkinfo == PATH_DIR and targetinfo == PATH_FILE:
		target = path.join(link, path.basename(target))
	target = path.abspath(target)
	print("Creating a symbolic link at", link, "pointing to", target)
	Path(link).symlink_to(target)


def run_cmd(cmd, print_output=True, realtime=False, print_command=False):
	if print_command:
		print(" ".join(cmd))
	proc = subprocess.Popen(cmd,
		stdout=subprocess.PIPE,
		stderr=subprocess.STDOUT)
	output = ""
	status = 0
	if realtime:  # print the command's output in real time, ignores print_output
		while True:
			try:
				realtime_output = proc.stdout.readline().decode("utf-8")
				if realtime_output == "" and status is not None:
					return ("", status)
				if realtime_output:
					print(realtime_output.strip())
					output += realtime_output
				status = proc.poll()
			except KeyboardInterrupt:
				return (output, 0)
	else:  # wait until the command is finished to print the output
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


def set_vars(goos=""):
	""" Sets GOOS-related variables to be used globally"""
	global gcos
	global gcos_name  # used for release, since macOS GOOS is "darwin"
	global exe
	global gochan_bin
	global gochan_exe
	global migration_bin
	global migration_exe

	if goos != "":
		os.environ["GOOS"] = goos

	gcos, gcos_status = run_cmd(("go", "env", "GOOS"), print_output=False)
	exe, exe_status = run_cmd(("go", "env", "GOEXE"), print_output=False)
	if gcos_status + exe_status != 0:
		print("Invalid GOOS value, check your GOOS environment variable")
		sys.exit(1)
	gcos_name = gcos
	if gcos_name == "darwin":
		gcos_name = "macos"

	gochan_bin = "gochan"
	gochan_exe = "gochan" + exe
	migration_bin = "gochan-migration"
	migration_exe = "gochan-migration" + exe


def build(debugging=False, plugin_path="", static_templates=False):
	"""Build the gochan executable for the current GOOS"""
	pwd = os.getcwd()
	trimpath = f"-trimpath={pwd}"

	gcflags_debug = " -l -N" if debugging else ""
	gcflags = f"-gcflags={trimpath}{gcflags_debug}"
	ldflags_debug = "" if debugging else " -w -s"
	ldflags = f"-ldflags=-X main.versionStr={GOCHAN_VERSION} -X main.dbVersionStr={DATABASE_VERSION} {ldflags_debug}"
	build_cmd_base = ["go", "build", "-v", "-trimpath", gcflags, ldflags]

	if static_templates:
		print("Building error pages from templates")
		with open("templates/404.html", "r") as tmpl404:
			tmpl404str = tmpl404.read().strip()
			with open("html/error/404.html", "w") as page404:
				page404.write(tmpl404str.format(GOCHAN_VERSION))
		with open("templates/5xx.html", "r") as tmpl5xx:
			tmpl5xxStr = tmpl5xx.read().strip()
			with open("html/error/500.html", "w") as page500:
				page500.write(tmpl5xxStr.format(version=GOCHAN_VERSION, title="Error 500: Internal Server error"))
			with open("html/error/502.html", "w") as page502:
				page502.write(tmpl5xxStr.format(version=GOCHAN_VERSION, title="Error 502: Bad gateway"))

	if debugging:
		print(f"Building for {gcos} with debugging symbols")
	else:
		print(f"Building for {gcos}")

	status = -1
	if plugin_path != "" and plugin_path is not None:
		build_cmd_base += ["-buildmode=plugin", plugin_path]
		status = run_cmd(build_cmd_base, realtime=True, print_command=True)[1]
		if status != 0:
			print(f"Failed building plugin at {plugin_path}, see output for details")
			sys.exit(1)
		print("Built plugin successfully")
		return

	gochan_build_cmd = build_cmd_base + ["-o", gochan_exe, "./cmd/gochan"]
	status = run_cmd(gochan_build_cmd, realtime=True, print_command=True)[1]
	if status != 0:
		print("Failed building gochan, see command output for details")
		sys.exit(1)
	print("Built gochan successfully")

	gochan_migrate_build_cmd = build_cmd_base + ["-o", migration_exe, "./cmd/gochan-migration"]
	status = run_cmd(gochan_migrate_build_cmd, realtime=True, print_command=True)[1]
	if status != 0:
		print("Failed building gochan-migration, see command output for details")
		sys.exit(1)
	print("Built gochan-migration successfully")


def clean():
	print("Cleaning up")
	del_files = ("gochan", "gochan.exe", "gochan-migration", "gochan-migration.exe", "releases/")
	for del_file in del_files:
		delete(del_file)


def install(prefix="/usr", document_root="/srv/gochan", symlinks=False, js_only=False, css_only=False, templates_only=False):
	if gcos == "windows":
		print("Installation is not currently supported for Windows, use the respective directory created by running `python build.py release`")
		sys.exit(1)
	mkdir(document_root)
	mkdir(path.join(prefix, "share/gochan"))
	print("Creating symbolic links: ", symlinks)

	start_dir = path.abspath(path.curdir)
	done = False
	if js_only is True:
		# args contains --js, install the JavaScript files
		os.chdir(path.join(start_dir,"html/"))
		if symlinks:
			symlink("js", path.join(document_root, "js"))
		else:
			copy("js/", document_root)
		os.chdir(start_dir)
		done = True
		print("JavaScript files installed")
	if css_only is True:
		# args contains --js, install the CSS files
		os.chdir(path.join(start_dir,"html/"))
		if symlinks:
			symlink("css/", path.join(document_root, "css"))
		else:
			copy("css/", document_root)
		os.chdir(start_dir)
		done = True
		print("CSS files installed")
	if templates_only is True:
		# args contains --js, install the templates
		os.chdir(start_dir)
		if symlinks:
			symlink("templates/", path.join(prefix, "share/gochan/templates"))
		else:
			copy("templates/", path.join(prefix, "share/gochan"))
		mkdir(path.join(prefix, "share/gochan/templates/override/"))
		done = True
		print("Templates installed")
	if done is True:
		print("Done installing specific stuff")
		return

	mkdir("/etc/gochan")
	mkdir("/var/log/gochan")

	for file in release_files:
		try:
			if file.startswith("html/"):
				trimmed = path.relpath(file, "html/")
				os.chdir(path.join(start_dir, "html/"))
				print("copying", trimmed,"to", path.join(document_root, trimmed))
				copy(trimmed, document_root)
				os.chdir(start_dir)
			else:
				os.chdir(start_dir)
				copy(file, path.join(prefix, "share/gochan"))
				mkdir(path.join(prefix, "share/gochan/templates/override/"))
		except shutil.SameFileError as err:
			print(err, "(skipping)")
		except FileNotFoundError:
			if file == "html/js/":
				print("Missing html/js directory, this must be built before installation by running python3 build.py js, or mkdir html/js if you don't want JavaScript")
			else:
				traceback.print_exc()
			sys.exit(1)


	if path.exists(gochan_exe) is False:
		build()
	print("Installing", gochan_exe, "to", path.join(prefix, "bin", gochan_exe))
	try:
		if symlinks:
			symlink(gochan_exe, path.join(prefix, "bin", gochan_exe))
		else:
			copy(gochan_exe, path.join(prefix, "bin", gochan_exe))
	except shutil.SameFileError:
		print(gochan_exe, "and", path.join(prefix, "bin", gochan_exe), "are the same file, skipping")

	if path.exists(migration_exe) is False:
		build()
	print("Installing ", migration_exe, "to", path.join(prefix, "bin", migration_exe))
	try:
		if symlinks:
			symlink(migration_exe, path.join(prefix, "bin", migration_exe))
		else:
			copy(migration_exe, path.join(prefix, "bin", migration_exe))
	except shutil.SameFileError:
		print(migration_exe, "and", path.join(prefix, "bin", migration_exe), "are the same file, skipping")

	print(
		"gochan was successfully installed. If you haven't already, you should copy\n",
		"examples/configs/gochan.example.json to /etc/gochan/gochan.json (modify as needed)\n",
		"You may also need to go to https://yourgochansite/manage/rebuildall to rebuild the javascript config")
	if gcos == "linux":
		print(
			"If your Linux distribution has systemd, you will also need to run the following commands:\n",
			"cp examples/configs/gochan-[mysql|postgresql|sqlite3].service /lib/systemd/system/gochan.service\n",
			"systemctl daemon-reload\n",
			"systemctl enable gochan.service\n",
			"systemctl start gochan.service")
	print("")


def js(watch=False):
	print("Transpiling JS")
	mkdir("html/js/")
	delete("html/js/gochan.js")
	delete("html/js/gochan.js.map")
	npm_cmd = ["npm", "--prefix", "frontend/", "run", "watch-ts" if watch else "build-ts"]
	status = run_cmd(npm_cmd, True, True, True)[1]
	if status != 0:
		print("JS transpiling failed with status", status)
		sys.exit(status)


def eslint(fix=False):
	print("Running eslint")
	npm_cmd = ["npm", "--prefix", "frontend/", "run", "eslint"]
	if fix:
		npm_cmd += ["--fix"]

	status = run_cmd(npm_cmd, True, True, True)[1]
	if status != 0:
		print("ESLint failed with status", status)
		sys.exit(status)


def release(goos):
	set_vars(goos)
	build(False, static_templates=True)
	release_name = gochan_bin + "-v" + GOCHAN_VERSION + "_" + gcos_name
	release_dir = path.join("releases", release_name)
	delete(release_dir)
	print("Creating release for", gcos_name, "\n")


	mkdir(path.join(release_dir, "html"))
	mkdir(path.join(release_dir, "sql"))
	for file in release_files:
		srcinfo = pathinfo(file)
		if srcinfo == PATH_NOTHING:
			raise FileNotFoundError(errno.ENOENT, os.strerror(errno.ENOENT), file)
		if srcinfo & PATH_FILE > 0:
			shutil.copy(file, path.join(release_dir, file))
		if srcinfo & PATH_DIR > 0:
			shutil.copytree(file, path.join(release_dir, file))
	copy(gochan_exe, path.join(release_dir, gochan_exe))
	copy(migration_exe, path.join(release_dir, migration_exe))
	delete(path.join(release_dir, "html/js/consts.js"))
	archive_type = "zip" if goos in ('windows', 'darwin') else "gztar"
	shutil.make_archive(release_dir, archive_type, root_dir="releases", base_dir=release_name)


def sass(watch=False):
	npm_cmd = ["npm", "--prefix", "frontend/", "run", "watch-sass" if watch else "build-sass"]
	status = run_cmd(npm_cmd, True, True, True)[1]
	if status != 0:
		print("Failed running sass with status", status)
		sys.exit(status)

def test(verbose=False, coverage=False):
	pkgs = os.listdir("pkg")
	for pkg in pkgs:
		cmd = ["go", "test"]
		if verbose:
			cmd += ["-v"]
		if coverage:
			cmd += ["-cover"]
		cmd += [path.join("./pkg", pkg)]
		run_cmd(cmd, realtime=True, print_command=True)


if __name__ == "__main__":
	action = "build"
	try:
		action = sys.argv.pop(1)
	except IndexError:  # no argument was passed
		pass
	if action.startswith("-") is False:
		sys.argv.insert(1, action)
	set_vars()

	valid_actions = (
		"build", "clean", "install", "js", "release", "sass", "test", "selenium"
	)
	parser = argparse.ArgumentParser(description="gochan build script")
	parser.add_argument("action", nargs=1, default="build", choices=valid_actions)
	if action in ('--help', '-h'):
		parser.print_help()
		sys.exit(2)

	if action == "build":
		parser.add_argument("--debug",
			help="build gochan and gochan-migrate with debugging symbols",
			action="store_true")
		parser.add_argument("--plugin",
			help="if used, builds the gochan-compatible Go plugin at the specified directory")
		parser.add_argument("--static-templates",
			help="if used, also (re)builds the static error page templates based on the current gochan version",
			action="store_true")
		args = parser.parse_args()
		build(args.debug, args.plugin, args.static_templates)
	elif action == "clean":
		clean()
		sys.exit(0)
	elif action == "install":
		parser.add_argument("--js",
			action="store_true",
			help="only install JavaScript (useful for frontend development)")
		parser.add_argument("--css",
			action="store_true",
			help="only install CSS")
		parser.add_argument("--templates",
			action="store_true",
			help="install the template files")
		parser.add_argument("--prefix",
			default="/usr",
			help="install gochan to this directory and its subdirectories")
		parser.add_argument("--documentroot",
			default="/srv/gochan",
			help="install files in ./html/ to this directory to be requested by a browser")
		parser.add_argument("--symlinks",
			action="store_true",
			help="create symbolic links instead of copying the files (may require admin/root privileges)")
		args = parser.parse_args()
		install(args.prefix, args.documentroot, args.symlinks, args.js, args.css, args.templates)
	elif action == "js":
		parser.add_argument("--watch", "-w",
			action="store_true",
			help="automatically rebuild when you change a file (keeps running)")
		parser.add_argument(
			"--eslint",
			action="store_true",
			help="Run eslint on the JavaScript code to check for possible problems")
		parser.add_argument("--eslint-fix",
			action="store_true",
			help="Run eslint on the JS code to try to fix detected problems")
		args = parser.parse_args()
		if args.eslint or args.eslint_fix:
			eslint(args.eslint_fix)
		else:
			js(args.watch)
	elif action == "release":
		parser.add_argument("--all", "-a",
			help="build releases for Windows, macOS, and Linux",
			action="store_true")
		args = parser.parse_args()
		mkdir("releases")
		if args.all:
			release("windows")
			release("darwin")
			release("linux")
		else:
			release(gcos)
	elif action == "sass":
		parser.add_argument("--watch", "-w",
			action="store_true",
			help="automatically rebuild when you change a file (keeps running)")

		args = parser.parse_args()
		sass(args.watch)
	elif action == "selenium":
		from tools.selenium_testing.runtests import setup_selenium_args, start_tests, close_tests
		args = setup_selenium_args(parser)
		try:
			start_tests(args.__dict__)
		except KeyboardInterrupt:
			print("Tests interrupted by KeyboardInterrupt, exiting")
		except Exception:
			traceback.print_exc()
			close_tests()
	elif action == "test":
		parser.add_argument("--verbose","-v",
			action="store_true",
			help="Print log messages in the tests")
		parser.add_argument("--coverage",
			action="store_true",
			help="Print unit test coverage")
		args = parser.parse_args()
		test(args.verbose, args.coverage)

	args = parser.parse_args()
