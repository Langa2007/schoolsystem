import { pendingOperations, removeOperation, type PendingOperation } from "./indexedDb";

type SyncResult = { status: "applied" | "conflict"; operationId: string };

export async function syncPending(apiBase: string, fetcher: typeof fetch = fetch): Promise<SyncResult[]> {
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
    const result = (await response.json()) as SyncResult;
    results.push(result);
    if (result.status === "applied") {
      await removeOperation(operation.id);
    }
  }
  return results;
}

export function registerBackgroundSync(): void {
  if ("serviceWorker" in navigator) {
    void navigator.serviceWorker.register("/service-worker.js");
  }
}
