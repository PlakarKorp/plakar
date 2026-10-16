import prettyBytes from "pretty-bytes";

export function Size(size: number) {
  return {
    Value: size,
    Pretty: prettyBytes(size),
    // Called by JSON.stringify to convert the object back to a string
    toJSON() {
      return size;
    },
  };
}
