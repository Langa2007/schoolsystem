import apiClient from "./apiClient.js";

export const submitPrefectCount = (payload) => apiClient.post("/api/v1/attendance/prefect-count", payload);
export const postMorningRoll = (payload) => apiClient.post("/api/v1/attendance/morning-roll", payload);
export const recordLesson = (payload) => apiClient.post("/api/v1/attendance/lessons", payload);
export const reconcileAttendance = (payload) => apiClient.post("/api/v1/attendance/reconcile", payload);
export const getAttendanceSummary = (params) => apiClient.get("/api/v1/attendance/summary", { params });
