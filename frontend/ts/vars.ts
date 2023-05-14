import jquery from "jquery";

export default (window.$ = window.jQuery = jquery);

// overwrite jQuery's deferred exception hook, because otherwise the sourcemap
// is useless if AJAX is involved
jquery.Deferred.exceptionHook = function(err: any) {
	// throw err;
	return err;
};


export const downArrow = "&#9660;";
export const upArrow = "&#9650;";
