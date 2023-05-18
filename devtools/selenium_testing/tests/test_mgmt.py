import unittest

from selenium.webdriver.common.by import By
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support.select import Select

from ..options import TestingOptions
from ..util.posting import make_post
from ..util.manage import staff_login

options: TestingOptions = None

def add_manage_tests(use_options: TestingOptions, suite: unittest.TestSuite):
	global options
	options = use_options
	suite.addTest(unittest.defaultTestLoader.loadTestsFromTestCase(TestManageActions))

class TestManageActions(unittest.TestCase):
	def test_login(self):
		staff_login(options)
		self.assertEqual(
			options.driver.find_element(by=By.CSS_SELECTOR, value="header h1").text,
			"Dashboard",
			"Testing staff login")

	def test_makeBoard(self):
		if options.board_exists("seleniumtesting"):
			raise Exception("Board /seleniumtests/ already exists")
		staff_login(options)
		options.goto_page("manage/boards")

		# fill out the board creation form
		options.driver.find_element(by=By.NAME, value="dir").\
			send_keys("seleniumtesting")
		options.driver.find_element(by=By.NAME, value="title").\
			send_keys("Selenium testing")
		options.driver.find_element(by=By.NAME, value="subtitle").\
			send_keys("Board for testing Selenium")
		options.driver.find_element(by=By.NAME, value="description").\
			send_keys("Board for testing Selenium")
		options.driver.find_element(by=By.NAME, value="docreate").click()
		options.driver.switch_to.alert.accept()
		WebDriverWait(options.driver, 10).until(
			EC.presence_of_element_located((
				By.CSS_SELECTOR,
				'div#topbar a[href="/seleniumtesting/"]')))

		make_post(options, "seleniumtesting", self)

		options.goto_page("manage/boards")
		sel = Select(options.driver.find_element(by=By.ID, value="modifyboard"))
		sel.select_by_visible_text("/seleniumtesting/ - Selenium testing")
		options.driver.find_element(by=By.NAME, value="dodelete").click()
		options.driver.switch_to.alert.accept()
		WebDriverWait(options.driver, 10).until_not(
			EC.presence_of_element_located((
				By.CSS_SELECTOR,
				'div#topbar a[href="/seleniumtesting/"]')))
