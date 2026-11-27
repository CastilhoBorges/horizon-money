// Conventional Commits, checked locally by the commit-msg hook and in CI on the PR title.
export default {
  extends: ["@commitlint/config-conventional"],
  rules: {
    // Long bodies are fine (e.g. wrapped explanations, co-author trailers).
    "body-max-line-length": [0],
    "footer-max-line-length": [0],
  },
};
