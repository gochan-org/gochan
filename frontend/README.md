# gochan.js development

## Building
You can technically use the npm build script directly for building gochan.js, but it's more convenient to just run `./build.py js` from the gochan repo root directory.

If you want to build it without minification, run `./build.py js --nominify`. If you want to have it watch the JS files for changes and rebuild them when you make any in realtime, use the `--watch` flag.

To install your gochan.js after building it, run `./build.py install --js`.

## Testing
Gochan unit testing with [Jest](https://jestjs.io) is still in its early stages and can be run by calling `npm run test` from the frontend directory.

Depending on your npm version, you may need to run this if you have the most up to date npm version available in your distro's repo but still get an error saying something like "Missing required argument #1" when you run `npm install`.
```
sudo npm install -g n
sudo n latest
sudo npm install -g npm
npm install
```