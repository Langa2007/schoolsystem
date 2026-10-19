import { pendingOperations, removeOperation } from "./indexedDb.js";

export async function syncPending(apiBase, fetcher = fetch) {
  const results: SyncResult[] = [];
  for (const operation of await pendingOperations()) {
    const response = await fetcher(`${apiBase}/api/v1/sync/operations`, {
      method: "POST",
      headers: { "Content-Type": "application/json", "Idempotency-Key": operation.id },
      body: JSON.stringify(operation),
    });
    if (!response.ok) {
      throw new Error(`sync failed with HTTP ${response.status}`);
    }
    const result = await response.json();
    results.push(result);
    if (result.status === "applied") {
      await removeOperation(operation.id);
    }
  }
  return results;
}

export function registerBackgroundSync() {
  if ("serviceWorker" in navigator) {
    void navigator.serviceWorker.register("/service-worker.js");
  }
}
