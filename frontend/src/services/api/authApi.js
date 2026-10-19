import apiClient from "./apiClient.js";

export const login = (credentials) => apiClient.post("/api/v1/auth/login", credentials);
export const getCurrentUser = () => apiClient.get("/api/v1/auth/me");
export const changePassword = (passwords) => apiClient.post("/api/v1/auth/password", passwords);
