import globals from "globals";
import prettierConfig from "eslint-config-prettier";

export default [
  {
    ignores: ["wailsjs/**", "**/node_modules/**", "../build/**"],
  },
  {
    files: ["src/**/*.js"],
    languageOptions: {
      ecmaVersion: 2022,
      sourceType: "module",
      globals: {
        ...globals.browser,
        ...globals.es2021,
        // Wails specific globals
        go: "readonly",
        runtime: "readonly"
      },
    },
    rules: {
      "no-unused-vars": ["warn", { "argsIgnorePattern": "^_" }],
      "no-undef": "error",
      "no-console": ["warn", { allow: ["warn", "error", "info"] }],
      "eqeqeq": ["error", "always", { "null": "ignore" }],
      "curly": "error",
    },
  },
  prettierConfig,
];
