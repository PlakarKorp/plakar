/**
 * This function is based on the type FileMode from the Go standard library, in io/fs/fs.go.
 */
export function Mode(mode: number) {
  // The commented modes below are not used in the Go standard library, but it
  // might be useful to uncomment them if needed in the future.

  const ModeDir = 1 << (32 - 1 - 0); // d: is a directory
  // const ModeAppend = 1 << (32 - 1 - 1); // a: append-only
  // const ModeExclusive = 1 << (32 - 1 - 2); // l: exclusive use
  // const ModeTemporary = 1 << (32 - 1 - 3); // T: temporary file; Plan 9 only
  const ModeSymlink = 1 << (32 - 1 - 4); // L: symbolic link
  const ModeDevice = 1 << (32 - 1 - 5); // D: device file
  const ModeNamedPipe = 1 << (32 - 1 - 6); // p: named pipe (FIFO)
  const ModeSocket = 1 << (32 - 1 - 7); // S: Unix domain socket
  const ModeSetuid = 1 << (32 - 1 - 8); // u: setuid
  const ModeSetgid = 1 << (32 - 1 - 9); // g: setgid
  const ModeCharDevice = 1 << (32 - 1 - 10); // c: Unix character device, when ModeDevice is set
  const ModeSticky = 1 << (32 - 1 - 11); // t: sticky
  const ModeIrregular = 1 << (32 - 1 - 12); // ?: non-regular file; nothing else is known about this file

  const ModePerm = 0o777; // Unix permission bits

  const ModeType =
    ModeDir |
    ModeSymlink |
    ModeNamedPipe |
    ModeSocket |
    ModeDevice |
    ModeCharDevice |
    ModeIrregular;

  return {
    Value: mode,

    // Called by JSON.stringify to convert the object back to a string
    toJSON() {
      return mode;
    },

    toString: () => {
      const str = "dalTLDpSugct?";
      const buf: string[] = [];
      let w = 0;

      for (let i = 0; i < str.length; i++) {
        if (mode & (1 << (32 - 1 - i))) {
          buf[w] = str[i] ?? "";
          w++;
        }
      }

      if (w === 0) {
        buf[w] = "-";
        w++;
      }

      const rwx = "rwxrwxrwx";
      for (let i = 0; i < rwx.length; i++) {
        if (mode & (1 << (9 - 1 - i))) {
          buf[w] = rwx[i] ?? "";
        } else {
          buf[w] = "-";
        }
        w++;
      }

      return buf.join("");
    },

    isDir: () => {
      return (mode & ModeDir) != 0;
    },

    isSymlink: () => {
      return (mode & ModeSymlink) != 0;
    },

    isRegular: () => {
      return (mode & ModeType) == 0;
    },

    isSetuid: () => {
      return (mode & ModeSetuid) != 0;
    },

    isSetgid: () => {
      return (mode & ModeSetgid) != 0;
    },

    isSticky: () => {
      return (mode & ModeSticky) != 0;
    },

    Perm() {
      return mode & ModePerm;
    },

    Type() {
      return mode & ModeType;
    },
  };
}
