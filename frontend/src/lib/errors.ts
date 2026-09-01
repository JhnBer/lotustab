export function isCanceled(err: unknown): boolean {
  if (typeof err === "object" && err !== null && "name" in err && (err as { name: string }).name === "CancelError") {
    return true;
  }
  const message = errorMessage(err).toLowerCase();
  return message.includes("context canceled") || message.includes("context cancelled");
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
