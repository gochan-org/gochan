import jquery from "jquery";
import path from "path-browserify";

export default (window.$ = window.jQuery = jquery);

// overwrite jQuery's deferred exception hook, because otherwise the sourcemap
// is useless if AJAX is involved
jquery.Deferred.exceptionHook = function(err: unknown) {
	// throw err;
	return err;
};


export const downArrow = "&#9660;";
export const upArrow = "&#9650;";

/**
 * Joins the given path parts with the configured web root at the start, and returns the resulting path.
 * @param parts The path parts to join.
 * @returns The joined path.
 * @example
 * webPath("static", "css", "style.css"); // returns "/static/css/style.css" if webroot is "/"
 * webPath("static", "css", "style.css"); // returns "/myapp/static/css/style.css" if webroot is "/myapp/"
 * webPath(); // returns webroot (or "/" if webroot is not set, presumably because consts.js hasn't been loaded yet)
 */
export function webPath(...parts: string[]): string {
	return path.join(webroot ?? "/", ...parts);
}