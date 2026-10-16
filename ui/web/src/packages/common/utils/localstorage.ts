export function removeLocalStorage(key: string): void {
  localStorage.removeItem(key);
}

export function setLocalStorage<T>(key: string, value: T): void {
  localStorage.setItem(key, JSON.stringify(value));
}

export function getLocalStorage<T>(key: string): T | null {
  const item = localStorage.getItem(key);
  if (!item) {
    return null;
  }
  try {
    return JSON.parse(item) as T;
  } catch {
    console.error("Failed to parse localStorage item");
    return null;
  }
}
