import urllib.parse

from selenium.webdriver.common.by import By
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support.select import Select

from . import SeleniumTestCase
from ..util.posting import make_post, delete_post
import random
from ..util.manage import staff_login


class TestManageActions(SeleniumTestCase):
	def setUp(self) -> None:
		staff_login(self.options)
		return super().setUp()

	def get_recent_post_link(self, msg_text: str):
		trs = self.driver.find_elements(by=By.CSS_SELECTOR, value="#content table tr")
		for tr in trs:
			tds = tr.find_elements(by=By.TAG_NAME, value="td")
			for c in range(len(tds)):
				if tds[c].text == msg_text:
					# found the post we made
					link = tds[c-2].find_element(by=By.LINK_TEXT, value="Post")
					return link

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
			self.driver.find_element(by=By.ID, value="board-title").\
				text, "Login", "At login page")

	def test_recentPosts(self):
		new_msg = f"test_recentPosts {random.randint(0, 9999)}"
		old_msg = self.options.message
		self.options.message = new_msg
		make_post(self.options, "test", self)
		self.options.message = old_msg
		staff_login(self.options)
		self.driver.find_element(by=By.LINK_TEXT, value="Recent posts").click()
		WebDriverWait(self.driver, 10).until(
			EC.url_contains("/manage/recentposts"))
		
		post_link = self.get_recent_post_link(new_msg)
		link_href = post_link.get_attribute("href")
		self.assertIsNotNone(post_link, "Found recent post in recent posts list")
		post_link.click()
		WebDriverWait(self.driver, 10).until(
			EC.url_contains(link_href)) # link_href should be something like "/seleniumtesting/ref/<threadOP>.html#<postID>"
		
		fragment = urllib.parse.urldefrag(self.driver.current_url).fragment
		delete_post(self.options, fragment, self.options.password)

		self.options.goto_page("/manage/recentposts")
		post_link = self.get_recent_post_link(new_msg)
		self.assertIsNone(post_link, "Confirmed that recent post was deleted")

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
