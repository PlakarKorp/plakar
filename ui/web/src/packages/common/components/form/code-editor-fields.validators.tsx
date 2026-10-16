import yaml from "yaml";

export function validateCodeEditorJSON({ value }: { value: string }) {
  if (!value.trim()) {
    return undefined;
  }
  try {
    JSON.parse(value);
    return undefined;
  } catch {
    return "Invalid JSON format";
  }
}

export function validateCodeEditorYAML({ value }: { value: string }) {
  if (!value.trim()) {
    return undefined;
  }
  try {
    yaml.parse(value);
    return undefined;
  } catch {
    return "Invalid YAML format";
  }
}
