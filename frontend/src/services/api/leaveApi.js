import apiClient from "./apiClient.js";

export const submitLeaveRequest = (payload) => apiClient.post("/api/v1/leave-requests", payload);
export const decideLeaveRequest = (id, decision) => apiClient.post(`/api/v1/leave-requests/${encodeURIComponent(id)}/decision`, decision);
