import apiClient from "./apiClient.js";

export const getTerms = (params) => apiClient.get("/api/v1/terms", { params });
export const createTerm = (term) => apiClient.post("/api/v1/terms", term);
export const getStreams = (params) => apiClient.get("/api/v1/streams", { params });
export const createStream = (stream) => apiClient.post("/api/v1/streams", stream);
