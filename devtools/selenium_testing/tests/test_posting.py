from os import path
import unittest

from selenium.webdriver.common.by import By
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.ui import WebDriverWait

from ..options import TestingOptions
from ..util.posting import make_post, delete_post, send_post, threadRE
from ..util import qr
from ..util.manage import staff_login

options: TestingOptions = None

def add_posting_tests(use_options: TestingOptions, suite: unittest.TestSuite):
	global options
	options = use_options
	suite.addTest(unittest.defaultTestLoader.loadTestsFromTestCase(TestPosting))

class TestPosting(unittest.TestCase):
	def test_qr(self):
		options.goto_page(options.board)
		elem = options.driver.find_element(by=By.ID, value="board-subtitle")
		self.assertIn("Board for testing stuff", elem.text)
		qr.openQR(options.driver)
		self.assertTrue(qr.qrIsVisible(options.driver),
			"Confirm that the QR box was properly opened")
		qr.closeQR(options.driver)
		self.assertFalse(qr.qrIsVisible(options.driver),
			"Confirm that the QR box was properly closed")

	def test_makeThread(self):
		make_post(options, options.board, self)

		threadID = threadRE.findall(options.driver.current_url)[0][1]
		cur_url = options.driver.current_url
		delete_post(options, int(threadID), "")
		WebDriverWait(options.driver, 10).until(
			EC.url_changes(cur_url))
		self.assertNotIn("Error :c", options.driver.title, "No errors when we try to delete the post we just made")

	def test_moveThread(self):
		if not options.board_exists("test2"):
			staff_login(options)
			options.goto_page("manage/boards")

			# fill out the board creation form
			options.driver.find_element(by=By.NAME, value="dir").send_keys("test2")
			options.driver.find_element(by=By.NAME, value="title").send_keys("Testing board #2")
			options.driver.find_element(by=By.NAME, value="subtitle").send_keys("Board for testing thread moving")
			options.driver.find_element(by=By.NAME, value="description").send_keys("Board for testing thread moving")
			options.driver.find_element(by=By.NAME, value="docreate").click()
			options.driver.switch_to.alert.accept()
			WebDriverWait(options.driver, 10).until(
				EC.presence_of_element_located((By.CSS_SELECTOR, 'div#topbar a[href="/test2/"]')))

		options.goto_page(options.board)
		WebDriverWait(options.driver, 10).until(
			EC.element_to_be_clickable((By.CSS_SELECTOR, "form#postform input[type=submit]")))

		form = options.driver.find_element(by=By.CSS_SELECTOR, value="form#postform")
		send_post(form,
			options.name,
			options.email,
			options.subject,
			options.message % options.driver.name,
			path.abspath(options.upload_path),
			options.password)
		WebDriverWait(options.driver, 10).until(
			EC.url_matches(threadRE))

		cur_url = options.driver.current_url
		threadID = threadRE.findall(cur_url)[0][1]
		options.driver.find_element(
			by=By.CSS_SELECTOR,
			value=("input#check%s"%threadID)).click()
		cur_url = options.driver.current_url
		options.driver.find_element(
			by=By.CSS_SELECTOR,
			value="input[name=move_btn]").click()
		# wait for response to move_btn
		WebDriverWait(options.driver, 10).until(
			EC.title_contains("Move thread #%s" % threadID))

		options.driver.find_element(
			by=By.CSS_SELECTOR,
			value="input[type=submit]").click()
		# wait for response to move request (domove=1)
		WebDriverWait(options.driver, 10).until(
			EC.url_matches(threadRE))

		self.assertEqual(
			options.driver.find_element(
				by=By.CSS_SELECTOR,
				value="h1#board-title").text,
			"/test2/ - Testing board #2",
			"Verify that we properly moved the thread to /test2/")

		form = options.driver.find_element(by=By.CSS_SELECTOR, value="form#postform")
		send_post(form,
			options.name,
			options.email,
			"",
			"Reply to thread after it was moved",
			path.abspath(options.upload_path),
			options.password)

		delete_post(options, int(threadID), "")
		WebDriverWait(options.driver, 10).until(
			EC.url_changes(cur_url))
		self.assertNotIn("Error :c", options.driver.title,
			"No errors when we try to delete the moved thread")

