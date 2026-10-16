export function Number(value: number) {
  const units = ["k", "M", "B"];
  const thresholds = [1_000, 1_000_000, 1_000_000_000];

  let pretty = `${value}`;

  for (let i = thresholds.length - 1; i >= 0; i--) {
    const threshold = thresholds[i];
    if (value >= threshold) {
      const num = value / threshold;
      pretty = `${num.toFixed(1).replace(/\.0$/, "")}${units[i]}`;
      break;
    }
  }

  return {
    Value: value,
    Pretty: pretty,
    // Called by JSON.stringify to convert the object back to a string
    toJSON() {
      return value;
    },
  };
}
