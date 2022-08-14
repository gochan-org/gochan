/**
 * Formats the timestamp strings from JSON into a more readable format
 * @param {string} dateStr timestamp string, assumed to be in ISO Date-Time format
 */
export function formatDateString(dateStr) {
	let date = new Date(dateStr);
	return date.toDateString() + ", " + date.toLocaleTimeString();
}

/**
 * Formats the given number of bytes into an easier to read filesize
 * @param {number} size
 */
export function formatFileSize(size) {
	if(size < 1000) {
		return `${size} B`;
	} else if(size <= 100000) {
		return `${(size/1024).toFixed(1)} KB`;
	} else if(size <= 100000000) {
		return `${(size/1024/1024).toFixed(2)} MB`;
	}
	return `${(size/1024/1024/1024).toFixed(2)} GB`;
}