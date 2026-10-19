import apiClient from "./apiClient.js";

export const enrollInPathway = (payload) => apiClient.post("/api/v1/cbc/enrollments", payload);
export const recordAssessment = (payload) => apiClient.post("/api/v1/cbc/assessments", payload);
