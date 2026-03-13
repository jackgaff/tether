import { useEffect, useState } from "react";

export function useStoredString(key: string) {
  const [value, setValue] = useState(() => {
    if (typeof window === "undefined") {
      return "";
    }

    return window.localStorage.getItem(key) ?? "";
  });

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    if (value) {
      window.localStorage.setItem(key, value);
    } else {
      window.localStorage.removeItem(key);
    }
  }, [key, value]);

  return [value, setValue] as const;
}
