import apiClient from "./apiClient.js";

export const getSchoolSettings = () => apiClient.get("/api/v1/settings");
