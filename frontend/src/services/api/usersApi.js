import apiClient from "./apiClient.js";

export const getStudents = (params) => apiClient.get("/api/v1/students", { params });
export const registerStudent = (student) => apiClient.post("/api/v1/students", student);
export const registerStaff = (staff) => apiClient.post("/api/v1/staff", staff);
export const provisionUser = (user) => apiClient.post("/api/v1/users", user);
