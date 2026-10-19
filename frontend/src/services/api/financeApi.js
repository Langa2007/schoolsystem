import apiClient from "./apiClient.js";

export const createVoteHead = (payload) => apiClient.post("/api/v1/finance/vote-heads", payload);
export const createInvoice = (payload) => apiClient.post("/api/v1/finance/invoices", payload);
export const getStudentLedger = (studentId) => apiClient.get(`/api/v1/finance/ledger/${encodeURIComponent(studentId)}`);
export const reverseLedgerEntry = (payload) => apiClient.post("/api/v1/finance/ledger/reverse", payload);
export const setFeeWithholding = (payload) => apiClient.post("/api/v1/finance/fee-withholding", payload);
export const grantFeeWaiver = (payload) => apiClient.post("/api/v1/finance/waivers", payload);
export const getResultsAccess = (studentId) => apiClient.get(`/api/v1/finance/results-access/${encodeURIComponent(studentId)}`);
export const initiateMpesaStkPush = (payload) => apiClient.post("/api/v1/finance/mpesa/stk-push", payload);
export const getUnmatchedMpesaPayments = () => apiClient.get("/api/v1/finance/mpesa/unmatched");
export const resolveUnmatchedMpesaPayment = (payload) => apiClient.post("/api/v1/finance/mpesa/unmatched/resolve", payload);
