// @ts-check

import {
	defineConfig,
	globalIgnores,
} from "eslint/config";

import tsParser from "@typescript-eslint/parser";
import tseslint from 'typescript-eslint';
import js from "@eslint/js";

export default defineConfig([{
	languageOptions: {
		parser: tsParser,

		globals: {
			jest: "readonly",
		},

		"sourceType": "module",

		parserOptions: {
			"ecmaFeatures": {
				"experimentalObjectRestSpread": true,
			},
		},
	},

	extends: [js.configs.recommended, tseslint.configs.recommended],

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
			"varsIgnorePattern": "^_",
			"caughtErrorsIgnorePattern": "^_",
		}],

		"no-trailing-spaces": "warn",
	},
}, globalIgnores(["**/legacy/**/*", "webpack.config.js"])]);
