import browserify from "browserify";
import buffer from "vinyl-buffer"
import glob from "glob";
import gulp from "gulp";
import stdio from "stdio";
import uglify from "gulp-uglify";
import source from "vinyl-source-stream";
import watchify from "watchify";
import sourcemaps from 'gulp-sourcemaps';

const args = stdio.getopt({
	"watch": {key: "w", description: "Automatically rebuild when you change a file"},
	"minify": {key: "m", description: "minify generated gochan.js"}
});

let builder = browserify({
	entries: glob.sync("src/**/*.js")
});

function buildTask(minify) {
	console.log("Building gochan frontend");
	let babelOptions = {
		presets: ["@babel/preset-env"],
		comments: false
	}

	let out = builder.transform("babelify", babelOptions)
		.bundle()
		.pipe(source("gochan.js"))
		.pipe(buffer())
		.pipe(sourcemaps.init({ loadMaps: true }))
		
	if(minify)
		out = out.pipe(uglify());

	return out
		.pipe(sourcemaps.write('./maps/'))
		.pipe(gulp.dest("../html/js/", {overwrite: true}));
}

gulp.task("default", () => {
	if(args.watch) {
		builder = watchify(builder);
		builder.on("update", () => {
			return buildTask(args.minify)
		});
	}
	return buildTask(args.minify);
});
