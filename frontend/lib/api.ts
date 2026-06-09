const API_URL = process.env.NEXT_PUBLIC_API_URL!;
const ADMIN_API_URL = process.env.NEXT_PUBLIC_ADMIN_API_URL ?? API_URL;

export type ApiResponse<T = unknown> = {
  success: boolean;
  message: string;
  data: T;
  error?: {
    code: string;
    detail?: string;
  };
};

async function request<T>(
  path: string,
  options: RequestInit = {}
): Promise<ApiResponse<T>> {
  const baseURL = path.startsWith("/admin") ? ADMIN_API_URL : API_URL;
  const res = await fetch(`${baseURL}${path}`, {
    credentials: "include",
    headers: { "Content-Type": "application/json", ...options.headers },
    ...options,
  });

  // For PDF blob responses
  if (res.headers.get("content-type")?.includes("application/pdf")) {
    const blob = await res.blob();
    return { success: true, message: "ok", data: blob as T };
  }

  const json = await res.json();
  return json;
}

export const api = {
  get: <T>(path: string) => request<T>(path, { method: "GET" }),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "POST", body: JSON.stringify(body) }),
  put: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "PUT", body: JSON.stringify(body) }),
  delete: <T>(path: string) => request<T>(path, { method: "DELETE" }),
};
