import unittest

from selenium.webdriver.common.by import By
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support.select import Select

from . import SeleniumTestCase
from ..util.posting import make_post
from ..util.manage import staff_login


class TestManageActions(SeleniumTestCase):
	def test_login(self):
		staff_login(self.options)
		self.assertEqual(
			self.options.driver.find_element(by=By.CSS_SELECTOR, value="header h1").text,
			"Dashboard",
			"Testing staff login")

	def test_makeBoard(self):
		if self.options.board_exists("seleniumtesting"):
			raise Exception("Board /seleniumtests/ already exists")
		staff_login(self.options)
		self.options.goto_page("manage/boards")

		# fill out the board creation form
		self.options.driver.find_element(by=By.NAME, value="dir").\
			send_keys("seleniumtesting")
		self.options.driver.find_element(by=By.NAME, value="title").\
			send_keys("Selenium testing")
		self.options.driver.find_element(by=By.NAME, value="subtitle").\
			send_keys("Board for testing Selenium")
		self.options.driver.find_element(by=By.NAME, value="description").\
			send_keys("Board for testing Selenium")
		self.options.driver.find_element(by=By.NAME, value="docreate").click()
		self.options.driver.switch_to.alert.accept()
		WebDriverWait(self.options.driver, 10).until(
			EC.presence_of_element_located((
				By.CSS_SELECTOR,
				'div#topbar a[href="/seleniumtesting/"]')))

		make_post(self.options, "seleniumtesting", self)

		self.options.goto_page("manage/boards")
		sel = Select(self.options.driver.find_element(by=By.ID, value="modifyboard"))
		sel.select_by_visible_text("/seleniumtesting/ - Selenium testing")
		self.options.driver.find_element(by=By.NAME, value="dodelete").click()
		self.options.driver.switch_to.alert.accept()
		WebDriverWait(self.options.driver, 10).until_not(
			EC.presence_of_element_located((
				By.CSS_SELECTOR,
				'div#topbar a[href="/seleniumtesting/"]')))
