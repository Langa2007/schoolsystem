export type PendingOperation = {
  id: string;
  tenantId: string;
  actorId: string;
  entity: string;
  action: string;
  version: number;
  payload: unknown;
  createdAt: string;
};

const databaseName = "elimucore-offline";
const storeName = "operations";

export function openOfflineDb(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(databaseName, 1);
    request.onerror = () => reject(request.error ?? new Error("IndexedDB open failed"));
    request.onupgradeneeded = () => {
      const database = request.result;
      if (!database.objectStoreNames.contains(storeName)) {
        database.createObjectStore(storeName, { keyPath: "id" });
      }
    };
    request.onsuccess = () => resolve(request.result);
  });
}

export async function queueOperation(operation: PendingOperation): Promise<void> {
  const database = await openOfflineDb();
  await new Promise<void>((resolve, reject) => {
    const transaction = database.transaction(storeName, "readwrite");
    transaction.objectStore(storeName).put(operation);
    transaction.onerror = () => reject(transaction.error ?? new Error("IndexedDB write failed"));
    transaction.oncomplete = () => resolve();
  });
  database.close();
}

export async function pendingOperations(): Promise<PendingOperation[]> {
  const database = await openOfflineDb();
  const values = await new Promise<PendingOperation[]>((resolve, reject) => {
    const request = database.transaction(storeName, "readonly").objectStore(storeName).getAll();
    request.onerror = () => reject(request.error ?? new Error("IndexedDB read failed"));
    request.onsuccess = () => resolve(request.result as PendingOperation[]);
  });
  database.close();
  return values;
}

export async function removeOperation(id: string): Promise<void> {
  const database = await openOfflineDb();
  await new Promise<void>((resolve, reject) => {
    const transaction = database.transaction(storeName, "readwrite");
    transaction.objectStore(storeName).delete(id);
    transaction.onerror = () => reject(transaction.error ?? new Error("IndexedDB delete failed"));
    transaction.oncomplete = () => resolve();
  });
  database.close();
}
