export type UserRole = "employee" | "manager"

export interface AuthUser {
  id: string
  name: string
  email: string
  role: UserRole
  managerId: string | null
}

export interface AuthResponse {
  user: AuthUser
}

export interface ApiErrorResponse {
  error: {
    code: string
    message: string
  }
}

const networkErrorMessage =
  "サーバーへ接続できませんでした。しばらくしてからもう一度お試しください。"
const invalidResponseMessage =
  "サーバーから予期しない応答がありました。しばらくしてからもう一度お試しください。"

export class AuthApiError extends Error {
  constructor(
    public readonly code: string,
    message: string,
    public readonly status: number,
  ) {
    super(message)
    this.name = "AuthApiError"
  }
}

function getApiBaseUrl(): string {
  const value = process.env.NEXT_PUBLIC_API_BASE_URL?.trim()

  if (!value) {
    throw new AuthApiError(
      "configuration_error",
      "APIの接続先が設定されていません。",
      0,
    )
  }

  return value.replace(/\/+$/, "")
}

async function requestAuth(
  path: string,
  init?: RequestInit,
): Promise<Response> {
  const url = `${getApiBaseUrl()}${path}`

  try {
    return await fetch(url, {
      ...init,
      credentials: "include",
      cache: "no-store",
    })
  } catch {
    throw new AuthApiError("network_error", networkErrorMessage, 0)
  }
}

async function readJson(response: Response): Promise<unknown> {
  try {
    return await response.json()
  } catch {
    return null
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null
}

function isAuthUser(value: unknown): value is AuthUser {
  if (!isRecord(value)) {
    return false
  }

  return (
    typeof value.id === "string" &&
    typeof value.name === "string" &&
    typeof value.email === "string" &&
    (value.role === "employee" || value.role === "manager") &&
    (typeof value.managerId === "string" || value.managerId === null)
  )
}

function isAuthResponse(value: unknown): value is AuthResponse {
  return isRecord(value) && isAuthUser(value.user)
}

function isApiErrorResponse(value: unknown): value is ApiErrorResponse {
  if (!isRecord(value) || !isRecord(value.error)) {
    return false
  }

  return (
    typeof value.error.code === "string" &&
    typeof value.error.message === "string"
  )
}

async function readAuthResponse(response: Response): Promise<AuthResponse> {
  const body = await readJson(response)

  if (!response.ok) {
    if (isApiErrorResponse(body)) {
      throw new AuthApiError(
        body.error.code,
        body.error.message,
        response.status,
      )
    }

    throw new AuthApiError(
      response.status === 401 ? "unauthenticated" : "invalid_response",
      response.status === 401 ? "認証が必要です。" : invalidResponseMessage,
      response.status,
    )
  }

  if (!isAuthResponse(body)) {
    throw new AuthApiError("invalid_response", invalidResponseMessage, 0)
  }

  return body
}

export async function login(
  email: string,
  password: string,
): Promise<AuthUser> {
  const response = await requestAuth("/api/auth/login", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ email, password }),
  })

  return (await readAuthResponse(response)).user
}

export async function getCurrentUser(): Promise<AuthUser> {
  const response = await requestAuth("/api/auth/me")
  return (await readAuthResponse(response)).user
}

export async function logout(): Promise<void> {
  const response = await requestAuth("/api/auth/logout", {
    method: "POST",
  })

  if (response.ok) {
    return
  }

  const body = await readJson(response)
  if (isApiErrorResponse(body)) {
    throw new AuthApiError(body.error.code, body.error.message, response.status)
  }

  throw new AuthApiError(
    "invalid_response",
    invalidResponseMessage,
    response.status,
  )
}
