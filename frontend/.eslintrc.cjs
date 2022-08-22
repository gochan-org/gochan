module.exports = {
    "env": {
        "browser": true,
        "es2021": true
    },
    "extends": "eslint:recommended",
    "parserOptions": {
        "ecmaVersion": "latest",
        "sourceType": "module"
    },
    "rules": {
        "no-unused-vars": "warn",
        "semi": "warn",
        "no-constant-condition": "warn",
        "no-whitespace-before-property": "warn",
        "linebreak-style": ["error", "unix"],
        "brace-style": ["error", "1tbs"],
        "array-bracket-spacing": ["error", "never"],
        "block-spacing": ["error", "always"],
        "func-call-spacing": ["error", "never"],
        "space-before-blocks": ["warn", "always"],
        "no-undef": "error",
        "keyword-spacing": ["warn", {
            "overrides": {
                "if": {"after": false},
                "for": {"after": false},
                "catch": {"after": false},
                "switch": {"after": false},
                "while": {"after": false}
            }
        }]
    }
}
