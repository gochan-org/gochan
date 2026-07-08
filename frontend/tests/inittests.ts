// jQuery can't be imported in a non-browser environment (jest.config.ts), so we import it here after jsdom has been loaded
import jquery from "jquery";

(global as {
	jQuery?: typeof jquery;
 }).jQuery = jquery;

