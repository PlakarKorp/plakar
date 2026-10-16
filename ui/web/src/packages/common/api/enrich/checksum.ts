export function Checksum(value: string) {
  return {
    Value: value,
    toHumanString: () => {
      return value.slice(0, 8);
    },

    // Called by JSON.stringify to convert the object back to a string
    toJSON() {
      return value;
    },
  };
}
