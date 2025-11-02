import {
	defineConfig,
	globalIgnores,
} from "eslint/config";

import tsParser from "@typescript-eslint/parser";
import globals from "globals";
import typescriptEslint from "@typescript-eslint/eslint-plugin";
import js from "@eslint/js";

import {
	FlatCompat,
} from "@eslint/eslintrc";

const compat = new FlatCompat({
	baseDirectory: import.meta.dirname,
	recommendedConfig: js.configs.recommended,
	allConfig: js.configs.all
});

export default defineConfig([{
	languageOptions: {
		parser: tsParser,

		globals: {
			...globals.browser,
			...globals.jest,
			...globals.node,
		},

		"sourceType": "module",

		parserOptions: {
			"ecmaFeatures": {
				"experimentalObjectRestSpread": true,
				"jsx": true,
			},
		},
	},

	extends: compat.extends("eslint:recommended", "plugin:@typescript-eslint/recommended"),

	plugins: {
		"@typescript-eslint": typescriptEslint,
	},

	"rules": {
		"indent": ["warn", "tab"],
		"linebreak-style": ["error", "unix"],

		"quotes": ["warn", "double", {
			"allowTemplateLiterals": true,
			"avoidEscape": true,
		}],

		"semi": ["error", "always"],
		"no-var": ["error"],
		"brace-style": ["error"],
		"array-bracket-spacing": ["error", "never"],
		"block-spacing": ["error", "always"],
		"no-spaced-func": ["error"],
		"no-whitespace-before-property": ["error"],
		"space-before-blocks": ["error", "always"],

		"keyword-spacing": ["error", {
			"overrides": {
				"if": {
					"after": false,
				},

				"for": {
					"after": false,
				},

				"catch": {
					"after": false,
				},

				"switch": {
					"after": false,
				},

				"while": {
					"after": false,
				},
			},
		}],

		"no-constant-condition": ["off"],
		"eqeqeq": ["error"],
		"@typescript-eslint/no-explicit-any": ["off"],

		"@typescript-eslint/no-unused-vars": ["warn", {
			"argsIgnorePattern": "^_",
		}],

		"no-trailing-spaces": "warn",
	},
}, globalIgnores(["**/legacy/**/*", "webpack.config.js"])]);
