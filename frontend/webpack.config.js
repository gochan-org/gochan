const path = require('path');

module.exports = {
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
			"path": require.resolve("path-browserify")
		}
	},
	output: {
		filename: "gochan.js",
		path: path.resolve(__dirname, '../html/js/'),
	},
	devtool: "source-map",
	mode: "production"
};