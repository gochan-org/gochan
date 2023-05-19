#!/usr/bin/env python3

# Don't forget to set NewThreadDelay and ReplyDelay to 0 in
# the config (with gochan running) before running, otherwise some posts may be rejected
# by the server

import argparse
import unittest

from .options import TestingOptions
from .tests import SeleniumTestCase
from .tests.test_mgmt import TestManageActions
from .tests.test_posting import TestPosting

testingSite = "http://192.168.56.3"
testingBoard = "test"
options:TestingOptions = None

def start_tests(browser:str, headless=False, keep_open=False, site="", board="", upload="", single_test = ""):
	global options
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
	if single_test == "":
		suite = unittest.suite.TestSuite()
		SeleniumTestCase.add(suite, options, TestPosting)
		SeleniumTestCase.add(suite, options, TestManageActions)
		unittest.TextTestRunner(verbosity=3, descriptions=True).run(suite)
	else:
		import importlib.util
		
		rindex = -1
		try:
			rindex =  single_test.rindex(":")
		except ValueError:
			raise ValueError("Single test must be of the format /path/to/test.py:TestCaseClass")
		test_location = single_test[:rindex]
		test_class  = single_test[rindex+1:]
		if test_location == "" or test_class == "":
			raise ValueError("Single test must be of the format /path/to/test.py:TestCaseClass")
		print("Single test module location:", test_location)
		print("Single test class:", test_class)
		
		spec = importlib.util.spec_from_file_location(test_class, test_location)
		module = importlib.util.module_from_spec(spec)
		module.__package__ = "devtools.selenium_testing.tests"
		spec.loader.exec_module(module)
		
		suite = unittest.suite.TestSuite()
		SeleniumTestCase.add(suite, options, module.__dict__[test_class])
		unittest.TextTestRunner(verbosity=3, descriptions=True).run(suite)
	options.close()

def close_tests():
	if options != None:
		options.close()

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
		start_tests(args.browser, args.headless, args.keepopen, args.site, args.board, "", args.singletest)
	except KeyboardInterrupt:
		print("Tests interrupted by KeyboardInterrupt, exiting")
		close_tests()