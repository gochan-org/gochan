from typing import Type
import unittest

from ..options import TestingOptions

options: TestingOptions = None

class SeleniumTestCase(unittest.TestCase):
	@staticmethod
	def add(suite: unittest.TestSuite, use_options: TestingOptions, test_class: Type[unittest.TestCase]):
		global options
		options = use_options
		suite.addTest(unittest.defaultTestLoader.loadTestsFromTestCase(test_class))


	@property
	def options(self):
		return options


	@property
	def driver(self):
		if options is None:
			return None
		return options.driver
