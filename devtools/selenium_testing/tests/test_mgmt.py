import unittest

from selenium.webdriver.common.by import By
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support.select import Select

from . import SeleniumTestCase
from ..util.posting import make_post
import random
from ..util.manage import staff_login


class TestManageActions(SeleniumTestCase):
	def setUp(self) -> None:
		staff_login(self.options)
		return super().setUp()
	
	def test_login(self):
		self.assertEqual(
			self.driver.find_element(by=By.CSS_SELECTOR, value="header h1").text,
			"Dashboard",
			"Testing staff login")

	def test_logoutEverywhere(self):
		self.assertEqual(
			self.driver.find_element(by=By.CSS_SELECTOR, value="header h1").text,
			"Dashboard",
			"Testing staff login")

		logout_link = self.driver.find_element(by=By.LINK_TEXT, value="Log me out everywhere")
		logout_link.click()
		WebDriverWait(self.driver, 10).until(
			EC.presence_of_element_located((By.CSS_SELECTOR, 'input[value="Login"]')))
		self.assertEqual(
			self.driver.find_element(by=By.ID, value="board-title").text, "Login", "At login page")

	def test_recentPosts(self):
		new_msg = "test_recentPosts %d" % random.randint(0, 9999)
		old_msg = self.options.message
		self.options.message = new_msg
		make_post(self.options, "test", self)
		self.options.message = old_msg
		staff_login(self.options)
		self.driver.find_element(by=By.LINK_TEXT, value="Recent posts").click()
		tds = self.driver.find_elements(by=By.CSS_SELECTOR, value="#content table td")
		post_exists = False
		for td in tds:
			if td.text == new_msg:
				post_exists = True
		self.assertTrue(post_exists, "Found recent post in recent posts list")
		

	def test_makeBoard(self):
		if self.options.board_exists("seleniumtesting"):
			raise Exception("Board /seleniumtests/ already exists")
		self.options.goto_page("manage/boards")

		# fill out the board creation form
		self.driver.find_element(by=By.NAME, value="dir").\
			send_keys("seleniumtesting")
		self.driver.find_element(by=By.NAME, value="title").\
			send_keys("Selenium testing")
		self.driver.find_element(by=By.NAME, value="subtitle").\
			send_keys("Board for testing Selenium")
		self.driver.find_element(by=By.NAME, value="description").\
			send_keys("Board for testing Selenium")
		self.driver.find_element(by=By.NAME, value="docreate").click()
		self.driver.switch_to.alert.accept()
		WebDriverWait(self.driver, 10).until(
			EC.presence_of_element_located((
				By.CSS_SELECTOR,
				'div#topbar a[href="/seleniumtesting/"]')))

		make_post(self.options, "seleniumtesting", self)

		self.options.goto_page("manage/boards")
		sel = Select(self.driver.find_element(by=By.ID, value="modifyboard"))
		sel.select_by_visible_text("/seleniumtesting/ - Selenium testing")
		self.driver.find_element(by=By.NAME, value="dodelete").click()
		self.driver.switch_to.alert.accept()
		WebDriverWait(self.driver, 10).until_not(
			EC.presence_of_element_located((
				By.CSS_SELECTOR,
				'div#topbar a[href="/seleniumtesting/"]')))
