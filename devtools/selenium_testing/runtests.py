#!/usr/bin/env python3

# Don't forget to set NewThreadDelay and ReplyDelay to 0 in
# the config (with gochan running) before running, otherwise some posts may be rejected
# by the server

import argparse
import unittest

from .options import TestingOptions

from .tests.test_mgmt import add_manage_tests
from .tests.test_posting import add_posting_tests

testingSite = "http://192.168.56.3"
testingBoard = "test"


class SimpleTests(unittest.TestCase):
	def test_ok(self):
		self.assertEqual(1, 1, "1 = 1")

def startBrowserTests(browser:str, headless=False, keep_open=False, site="", board="", upload="", singleTest = ""):
	options = TestingOptions(browser, headless, keep_open)

	if headless:
		options.keep_open = False
	if site != "":
		options.site = site
	if board != "":
		options.board = board
	if upload != "":
		options.upload_path = upload

	print("Using browser %s (headless: %s) on site %s" % (browser, options.headless, options.site))
	suite = unittest.defaultTestLoader.loadTestsFromTestCase(SimpleTests)
	add_manage_tests(options, suite)
	add_posting_tests(options, suite)

	unittest.TextTestRunner(verbosity=3, descriptions=True).run(suite)
	if not options.keep_open:
		options.driver.close()


def parseArgs(argParser:argparse.ArgumentParser):
	testable_browsers = ("firefox","chrome","chromium", "edge")

	argParser.add_argument("--browser", choices=testable_browsers, required=True)
	argParser.add_argument("--site", default=testingSite,
		help=("Sets the site to be used for testing, defaults to %s" % testingSite))
	argParser.add_argument("--board", default=testingBoard,
		help="Sets the board to be used for testing")
	argParser.add_argument("--headless", action="store_true",
		help="If set, the driver will run without opening windows (overrides --keepopen if it is set)")
	argParser.add_argument("--keepopen", action="store_true",
		help="If set, the browser windows will not automatically close after the tests are complete")
	argParser.add_argument("--singletest", default="",
		help="If specified, only the test method with this name will be run")
	return argParser.parse_args()

if __name__ == "__main__":
	parser = argparse.ArgumentParser(description="Browser testing via Selenium")
	args = parseArgs(parser)
	try:
		startBrowserTests(args.browser, args.headless, args.keepopen, args.site, args.board)
	except KeyboardInterrupt:
		print("Tests interrupted, exiting")
