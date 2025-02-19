#!/usr/bin/env python3

# Use this to download the latest version of the pre-transpiled JavaScript files if you don't have Node/NPM installed
# it takes an optional command line argument, which specifies a custom output directory if set (js otherwise)

import gzip
from urllib.request import urlopen
import tarfile
import io
from os import path
import sys

VERSION = "v3.10.2"
DIR = f"gochan-{VERSION}_linux"
DOWNLOAD_URL = f"https://github.com/gochan-org/gochan/releases/download/{VERSION}/{DIR}.tar.gz"
JS_DIR = path.join(DIR, "html/js/")

if __name__ == "__main__":
	out_dir = "js"
	if len(sys.argv) == 2:
		match sys.argv[1]:
			case "-h" | "--help":
				print(f"usage: {sys.argv[0]} [path/to/out/js/]")
				sys.exit(0)
			case _:
				out_dir = sys.argv[1]

	with urlopen(DOWNLOAD_URL) as response:
		data = response.read()
		tar_bytes = gzip.decompress(data)
		buf = io.BytesIO(tar_bytes)
		with tarfile.open(fileobj=buf) as tar_file:
			files = tar_file.getmembers()
			for file in files:
				if file.path.startswith(JS_DIR):
					file.path = file.path[len(JS_DIR):]
					tar_file.extract(file, out_dir, filter=tarfile.tar_filter)
