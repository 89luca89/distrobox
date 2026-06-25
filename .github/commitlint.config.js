module.exports = {
  extends: ["@commitlint/config-conventional"],
  rules: {
    "body-max-line-length": [0, "always", Infinity], // disables the 100-char limit
    "header-max-length": [0, "always", Infinity], // disables the 100-char limit
  },
};
