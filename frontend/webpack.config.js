import path from "path";

export default {
	entry: './ts/gochan.ts',
	module: {
		rules: [{
			test: /\.tsx?$/,
			use: 'ts-loader',
			exclude: /node_modules/,
		}],
	},
	resolve: {
		extensions: ['.ts', '.js'],
		"fallback": {
			"path": import.meta.resolve("path-browserify")
		}
	},
	output: {
		filename: "gochan.js",
		path: path.resolve(import.meta.dirname, '../html/js/'),
	},
	devtool: "source-map",
	mode: "production"
};