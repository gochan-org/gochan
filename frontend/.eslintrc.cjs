module.exports = {
  "parser": "@typescript-eslint/parser",
  "env": {
    "browser": true,
    "jest": true,
    "node": true,
    "es6": true
  },
  "extends": [
    "eslint:recommended",
    "plugin:@typescript-eslint/recommended",
  ],
  "ignorePatterns": ["**/legacy/**"],
  "parserOptions": {
    "ecmaFeatures": {
      "experimentalObjectRestSpread": true,
      "jsx": true
    },
    "sourceType": "module"
  },
  "plugins": [
    "@typescript-eslint"
  ],
  "rules": {
    "indent": ["warn", "tab"],
    "linebreak-style": ["error", "unix"],
    "quotes": ["warn", "double", {
      "allowTemplateLiterals": true
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
        "if": {"after": false},
        "for": {"after": false},
        "catch": {"after": false},
        "switch": {"after": false},
        "while": {"after": false}
      }
    }],
    "no-constant-condition": ["off"],
    "@typescript-eslint/no-explicit-any": ["off"],
    "@typescript-eslint/no-unused-vars": ["warn", {
      "argsIgnorePattern": "^_"
    }],
  }
};
