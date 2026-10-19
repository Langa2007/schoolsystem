import apiClient from "./apiClient.js";

export const createExamSheet = (payload) => apiClient.post("/api/v1/exams/sheets", payload);
export const setMark = (sheetId, studentId, payload) => apiClient.put(`/api/v1/exams/sheets/${encodeURIComponent(sheetId)}/marks/${encodeURIComponent(studentId)}`, payload);
export const submitExamSheet = (sheetId) => apiClient.post(`/api/v1/exams/sheets/${encodeURIComponent(sheetId)}/submit`);
export const getVerificationList = (sheetId) => apiClient.get(`/api/v1/exams/sheets/${encodeURIComponent(sheetId)}/verification`);
