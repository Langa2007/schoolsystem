import axios from "axios";

export const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || "http://localhost:8080",
  headers: { "Content-Type": "application/json" },
});

apiClient.interceptors.request.use((config) => {
  const token = globalThis.localStorage?.getItem("elimucore.accessToken");
  const tenantId = globalThis.localStorage?.getItem("elimucore.tenantId");
  if (token) config.headers.Authorization = `Bearer ${token}`;
  if (tenantId) config.headers["X-Tenant-ID"] = tenantId;
  return config;
});

apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    const message = error.response?.data?.message || error.response?.data?.error || error.message || "Request failed";
    return Promise.reject(Object.assign(new Error(message), {
      status: error.response?.status,
      data: error.response?.data,
    }));
  },
);

export default apiClient;
