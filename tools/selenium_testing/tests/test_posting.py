from os import path

from selenium.webdriver.common.by import By
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.ui import WebDriverWait

from . import SeleniumTestCase
from ..util.posting import make_post, delete_post, send_post, threadRE
from ..util import qr
from ..util.manage import staff_login

class TestPosting(SeleniumTestCase):
	@classmethod
	def setUpClass(cls):
		super().setUpClass()


	def test_qr(self):
		self.options.goto_page(self.options.board1)
		elem = self.driver.find_element(by=By.ID, value="board-subtitle")
		self.assertIn("Board for testing stuff", elem.text)
		qr.openQR(self.driver)
		self.assertTrue(qr.qrIsVisible(self.driver),
			"Confirm that the QR box was properly opened")
		qr.closeQR(self.driver)
		self.assertFalse(qr.qrIsVisible(self.driver),
			"Confirm that the QR box was properly closed")

	def test_makeThread(self):
		make_post(self.options, self.options.board1, self)

		threadID = threadRE.findall(self.driver.current_url)[0][1]
		cur_url = self.driver.current_url
		delete_post(self.options, int(threadID), "")
		WebDriverWait(self.driver, 10).until(
			EC.url_changes(cur_url))
		self.assertNotIn("Error :c", self.driver.title, "No errors when we try to delete the post we just made")

	def test_moveThread(self):
		if not self.options.board_exists("test2"):
			staff_login(self.options)
			self.options.goto_page("manage/boards")

			# fill out the board creation form
			self.driver.find_element(by=By.NAME, value="dir").send_keys("test2")
			self.driver.find_element(by=By.NAME, value="title").send_keys("Testing board 2")
			self.driver.find_element(by=By.NAME, value="subtitle").send_keys("Board for testing thread moving")
			self.driver.find_element(by=By.NAME, value="description").send_keys("Board for testing thread moving")
			self.driver.find_element(by=By.NAME, value="docreate").click()
			self.driver.switch_to.alert.accept()
			WebDriverWait(self.driver, 10).until(
				EC.presence_of_element_located((By.CSS_SELECTOR, 'div#topbar a[href="/test2/"]')))

		self.options.goto_page(self.options.board1)
		WebDriverWait(self.driver, 10).until(
			EC.element_to_be_clickable((By.CSS_SELECTOR, "form#postform input[type=submit]")))

		form = self.driver.find_element(by=By.CSS_SELECTOR, value="form#postform")
		send_post(form,
			self.options.name,
			self.options.email,
			self.options.subject,
			self.options.message % self.driver.name,
			path.abspath(self.options.upload_path),
			self.options.password)
		WebDriverWait(self.driver, 10).until(
			EC.url_matches(threadRE))

		cur_url = self.driver.current_url
		threadID = threadRE.findall(cur_url)[0][1]
		self.driver.find_element(
			by=By.CSS_SELECTOR,
			value=("input#check%s"%threadID)).click()
		cur_url = self.driver.current_url
		self.driver.find_element(
			by=By.CSS_SELECTOR,
			value="input[name=move_btn]").click()
		# wait for response to move_btn
		WebDriverWait(self.driver, 10).until(
			EC.title_contains("Move thread #%s" % threadID))

		self.driver.find_element(
			by=By.CSS_SELECTOR,
			value="input[type=submit]").click()
		# wait for response to move request (domove=1)
		WebDriverWait(self.driver, 10).until(
			EC.url_matches(threadRE))

		self.assertEqual(
			self.driver.find_element(
				by=By.CSS_SELECTOR,
				value="h1#board-title").text,
			"/test2/ - Testing board 2",
			"Verify that we properly moved the thread to /test2/")

		form = self.driver.find_element(by=By.CSS_SELECTOR, value="form#postform")
		send_post(form,
			self.options.name,
			self.options.email,
			"",
			"Reply to thread after it was moved",
			path.abspath(self.options.upload_path),
			self.options.password)

		delete_post(self.options, int(threadID), "")
		WebDriverWait(self.driver, 10).until(
			EC.url_changes(cur_url))
		self.assertNotIn("Error :c", self.driver.title,
			"No errors when we try to delete the moved thread")

