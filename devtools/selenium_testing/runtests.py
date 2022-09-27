#!/usr/bin/env python3

# Don't forget to set NewThreadDelay and ReplyDelay to 0 in
# the config (with gochan running) before running, otherwise some posts may be rejected
# by the server

import argparse
from os import path
import re
import unittest
from urllib.parse import urljoin
from selenium import webdriver

from selenium.webdriver.chrome.webdriver import WebDriver
from selenium.webdriver.edge.webdriver import WebDriver
from selenium.webdriver.firefox.webdriver import WebDriver
from selenium.webdriver.chrome.options import Options  as ChromeOptions
from selenium.webdriver.edge.options import Options as EdgeOptions
from selenium.webdriver.firefox.options import Options as FirefoxOptions
from selenium.webdriver.common.by import By
from selenium.webdriver.remote.webelement import WebElement
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.ui import WebDriverWait

from .util.localstorage import LocalStorage
from .util.qr import openQR,closeQR,qrIsVisible

testingSite = "http://192.168.56.3"
testingName = "Selenium"
testingEmail = "selenium@gochan.org#noko"
testingMessage = "Hello, from Selenium!\n(driver is %s)"
testingSubject = "Selenium post creation"
testingUploadPath = "../../html/notbanned.png"
testingPassword = "12345"
testingBoard = "test"

threadRE = re.compile('.*/(\S+)/(\d+)(\+50)?.html')
browser = ""
headless = False
keepOpen = False

def gotoPage(driver: WebDriver, page: str):
	driver.get(urljoin(testingSite, page))

def loginToStaff(driver: WebDriver, username = "admin", pw = "password"):
	gotoPage(driver, "manage")
	driver.find_element(by=By.NAME, value="username").send_keys(username)
	driver.find_element(by=By.NAME, value="password").send_keys(pw)
	driver.find_element(by=By.CSS_SELECTOR, value="input[value=Login]").click()

def sendPost(postForm:WebElement, name="", email="", subject="", message="", file=""):
	postForm.find_element(by=By.NAME, value="postname").send_keys(name)
	postForm.find_element(by=By.NAME, value="postemail").send_keys(email)
	postForm.find_element(by=By.NAME, value="postsubject").send_keys(subject)
	postForm.find_element(by=By.NAME, value="postmsg").send_keys(message)
	if file != "":
		postForm.find_element(by=By.NAME, value="imagefile").send_keys(file)
	postForm.find_element(by=By.CSS_SELECTOR, value="input[type=submit]").click()

class TestRunner(unittest.TestCase):
	def setUp(self):
		if browser == "firefox":
			options = FirefoxOptions()
			if headless:
				options.headless = True
			self.driver = webdriver.Firefox(options=options)
		elif browser == "chrome" or browser == "chromium":
			options = ChromeOptions()
			if headless:
				options.headless = True
			if keepOpen:
				options.add_experimental_option("detach", True)
			self.driver = webdriver.Chrome(options=options)
		else:
			self.fail("Unrecognized --browser option '%s'" % browser)
		return super().setUp()

	def test_qr(self):
		gotoPage(self.driver, testingBoard)
		self.assertIn("/test/ - Testing board", self.driver.title)
		elem = self.driver.find_element(by=By.ID, value="board-subtitle")
		self.assertIn("Board for testing stuff", elem.text)
		openQR(self.driver)
		self.assertTrue(qrIsVisible(self.driver), "Confirm that the QR box was properly opened")
		closeQR(self.driver)
		self.assertFalse(qrIsVisible(self.driver), "Confirm that the QR box was properly closed")

	def test_localStorage(self):
		gotoPage(self.driver, testingBoard)
		localStorage = LocalStorage(self.driver)
		localStorage["foo"] = "bar"
		self.assertEqual(localStorage.get("foo"), "bar", "making sure that LocalStorage.get works")
		self.assertGreater(localStorage.__len__(), 0)

	def test_staffLogin(self):
		loginToStaff(self.driver)
		self.assertEqual(
			self.driver.find_element(by=By.CSS_SELECTOR, value="header h1").text,
			"Dashboard",
			"Testing staff login"
		)

	def test_makeThread(self):
		gotoPage(self.driver, testingBoard)
		WebDriverWait(self.driver, 10).until(
			EC.element_to_be_clickable((By.CSS_SELECTOR, "form#postform input[type=submit]")))
		
		valProp = self.driver.find_element(by=By.CSS_SELECTOR, value="form#postform input[type=submit]").get_property("value")
		self.assertEqual(valProp, "Post")
		form = self.driver.find_element(by=By.CSS_SELECTOR, value="form#postform")

		sendPost(form,
			testingName,
			testingEmail,
			testingSubject,
			testingMessage % self.driver.name,
			path.abspath(testingUploadPath))
		WebDriverWait(self.driver, 10).until(
			EC.url_matches(threadRE))
		threadID = threadRE.findall(self.driver.current_url)[0][1]
		self.driver.find_element(by=By.CSS_SELECTOR, value=("input#check%s"%threadID)).click()
		delPasswordInput = self.driver.find_element(by=By.CSS_SELECTOR, value="input#delete-password")
		val = delPasswordInput.get_attribute("value")
		if val == "" or val is None:
			val = testingPassword
			delPasswordInput.send_keys(testingPassword)
		cur_url = self.driver.current_url
		self.driver.find_element(by=By.CSS_SELECTOR, value="input[name=delete_btn]").click()
		self.driver.switch_to.alert.accept()
		WebDriverWait(self.driver, 10).until(
			EC.url_changes(cur_url))
		self.assertNotIn("Error :c", self.driver.title, "No errors when we try to delete the post we just made")

	def tearDown(self):
		if not keepOpen:
			self.driver.close()
		return super().tearDown()

def startBrowserTests(testBrowser:str, testHeadless=False, testKeepOpen=False, site=testingSite, board=testingBoard, upload=testingUploadPath, singleTest = ""):
	global browser
	global testingSite
	global testingBoard
	global testingUploadPath
	global headless
	global keepOpen
	browser = testBrowser
	headless = testHeadless
	keepOpen = testKeepOpen
	if headless:
		keepOpen = False
	testingSite = site
	testingBoard = board
	testingUploadPath = upload

	print("Using browser %s (headless: %s) on site %s" % (browser, headless, testingSite))
	suite:unittest.TestSuite = None
	if singleTest == "":
		suite = unittest.defaultTestLoader.loadTestsFromTestCase(TestRunner)
	else:
		suite = unittest.defaultTestLoader.loadTestsFromName(singleTest,TestRunner)
	unittest.TextTestRunner(verbosity=3, descriptions=True).run(suite)


def parseArgs(parser:argparse.ArgumentParser):
	testable_browsers = ("firefox","chrome","chromium")

	parser.add_argument("--browser", choices=testable_browsers, required=True)
	parser.add_argument("--site", default=testingSite, help=("Sets the site to be used for testing, defaults to %s" % testingSite))
	parser.add_argument("--board", default=testingBoard, help="Sets the board to be used for testing")
	parser.add_argument("--headless", action="store_true", help="If set, the driver will run without opening windows (overrides --keepopen if it is set)")
	parser.add_argument("--keepopen", action="store_true", help="If set, the browser windows will not automatically close after the tests are complete")
	parser.add_argument("--singletest", default="", help="If specified, only the test method with this name will be run")
	return parser.parse_args()

if __name__ == "__main__":
	parser = argparse.ArgumentParser(description="Browser testing via Selenium")
	args = parseArgs(parser)
	try:
		startBrowserTests(args.browser, args.headless, args.keepopen, args.site, args.board)
	except KeyboardInterrupt:
		print("Tests interrupted, exiting")