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

from util.localstorage import LocalStorage
from util.qr import openQR,closeQR,qrIsVisible

testingSite = "http://192.168.56.3"
testingName = "Selenium"
testingEmail = "selenium@gochan.org#noko"
testingMessage = "Hello, from Selenium!\n(driver is %s)"
testingSubject = "Selenium post creation"
testingUploadPath = "../../html/banned.png"
testingPassword = "12345"
testingBoard = "test"

browser = ""
threadRE = re.compile('.*/(\S+)/(\d+)(\+50)?.html')


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
			options.headless = True
			self.driver = webdriver.Firefox(options=options)
		elif browser == "chrome" or browser == "chromium":
			options = ChromeOptions()
			options.headless = True
			self.driver = webdriver.Chrome(options=options)
		else:
			self.fail("Unrecognized --browser option '%s'" % browser)
		print("Using browser %s on site %s" % (browser, testingSite))
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
		self.driver.close()
		
		return super().tearDown()

