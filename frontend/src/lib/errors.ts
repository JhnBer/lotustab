export function isCanceled(err: unknown): boolean {
  if (typeof err === "object" && err !== null) {
    if ("name" in err && (err as { name: string }).name === "CancelError") {
      return true;
    }
    if ("code" in err) {
      const code = (err as { code?: unknown }).code;
      if (code === 1317 || code === "1317" || code === "70100") {
        return true;
      }
    }
  }
  const message = errorMessage(err).toLowerCase();
  return (
    message.includes("context canceled") ||
    message.includes("context cancelled") ||
    message.includes("query execution was interrupted") ||
    /\b1317\b/.test(message) ||
    /\b70100\b/.test(message)
  );
}

export function errorMessage(err: unknown): string {
  if (err == null) {
    return "Unknown error";
  }
  if (typeof err === "string") {
    return err;
  }
  if (err instanceof Error && err.message) {
    return err.message;
  }
  if (typeof err === "object" && "message" in err) {
    const message = (err as { message?: unknown }).message;
    if (typeof message === "string" && message) {
      return message;
    }
  }
  try {
    return JSON.stringify(err);
  } catch {
    return String(err);
  }
}
