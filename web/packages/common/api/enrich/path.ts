export function Path(path: string, username: string) {
  return {
    Value: path,

    // Called by JSON.stringify to convert the object back to a string
    toJSON() {
      return path;
    },

    // Attempt to shorten the path by replacing the user's home directory with '~'.
    toHumanString: () => {
      const tests = [
        {
          regex: new RegExp(`^/home/${username}/`),
          replace: "~/",
        },
        {
          regex: new RegExp(`^/Users/${username}/`),
          replace: "~/",
        },
      ];

      for (const r of tests) {
        if (r.regex.test(path)) {
          return path.replace(r.regex, r.replace);
        }
      }
      return path;
    },
  };
}
