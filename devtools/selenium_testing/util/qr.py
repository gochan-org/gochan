from selenium.webdriver.remote.webdriver import WebDriver
from selenium.webdriver.common.by import By
from selenium.common.exceptions import WebDriverException

def qrIsVisible(driver: WebDriver):
	try:
		return driver.find_element(by=By.CSS_SELECTOR, value="div#qr-box").is_displayed()
	except WebDriverException:
		return False

def enableQR(driver: WebDriver):
	topbar = driver.find_element(by=By.ID, value="topbar")
	topbar.click()
	topbar.find_element(by=By.CSS_SELECTOR, value="a#settings").click()
	useqr = driver.find_element(by=By.CSS_SELECTOR, value="input#useqr")
	if not useqr.is_selected():
		useqr.click()
	driver.find_element(by=By.CSS_SELECTOR, value="a.lightbox-x").click()

def disableQR(driver: WebDriver):
	topbar = driver.find_element(by=By.ID, value="topbar")
	topbar.click()
	topbar.find_element(by=By.CSS_SELECTOR, value="a#settings").click()
	useqr = driver.find_element(by=By.CSS_SELECTOR, value="input#useqr")
	if useqr.is_selected():
		useqr.click()
	driver.find_element(by=By.CSS_SELECTOR, value="a#lightbox-x").click()

def openQR(driver: WebDriver):
	enableQR(driver)
	if qrIsVisible(driver):
		return
	body = driver.find_element(by=By.CSS_SELECTOR, value="body")
	body.click()
	body.send_keys("q")


def closeQR(driver: WebDriver):
	if not qrIsVisible(driver):
		return
	closeLink = driver.find_element(by=By.LINK_TEXT, value="X")
	driver.execute_script("arguments[0].click();", closeLink)
