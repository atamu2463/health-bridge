"use client"

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react"

import {
  AuthApiError,
  getCurrentUser,
  login as loginRequest,
  logout as logoutRequest,
  type AuthUser,
  type UserRole,
} from "@/lib/auth-api"

export type AuthStatus =
  "loading" | "authenticated" | "unauthenticated" | "error"

interface AuthContextValue {
  status: AuthStatus
  user: AuthUser | null
  errorMessage: string | null
  login: (
    email: string,
    password: string,
    expectedRole: UserRole,
  ) => Promise<AuthUser>
  logout: () => Promise<void>
  refreshCurrentUser: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

function safeErrorMessage(error: unknown): string {
  if (error instanceof AuthApiError) {
    return error.message
  }

  return "認証状態を確認できませんでした。しばらくしてからもう一度お試しください。"
}

/**
 * Provides session state and authentication actions to descendants.
 * Request versions prevent older responses from overwriting newer state.
 */
export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>("loading")
  const [user, setUser] = useState<AuthUser | null>(null)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const requestVersion = useRef(0)

  const refreshCurrentUser = useCallback(async () => {
    const version = ++requestVersion.current
    setStatus("loading")
    setErrorMessage(null)

    try {
      const currentUser = await getCurrentUser()
      if (requestVersion.current !== version) {
        return
      }

      setUser(currentUser)
      setStatus("authenticated")
    } catch (error) {
      if (requestVersion.current !== version) {
        return
      }

      setUser(null)
      if (error instanceof AuthApiError && error.status === 401) {
        setStatus("unauthenticated")
        return
      }

      setErrorMessage(safeErrorMessage(error))
      setStatus("error")
    }
  }, [])

  useEffect(() => {
    const version = ++requestVersion.current
    let active = true

    void getCurrentUser()
      .then((currentUser) => {
        if (!active || requestVersion.current !== version) {
          return
        }

        setUser(currentUser)
        setStatus("authenticated")
      })
      .catch((error: unknown) => {
        if (!active || requestVersion.current !== version) {
          return
        }

        setUser(null)
        if (error instanceof AuthApiError && error.status === 401) {
          setStatus("unauthenticated")
          return
        }

        setErrorMessage(safeErrorMessage(error))
        setStatus("error")
      })

    return () => {
      active = false
    }
  }, [])

  /** Authenticates the expected role, rechecking the actual session before rethrowing login errors. */
  const login = useCallback(
    async (email: string, password: string, expectedRole: UserRole) => {
      const version = ++requestVersion.current
      let authenticatedUser: AuthUser

      try {
        authenticatedUser = await loginRequest(email, password, expectedRole)
      } catch (error) {
        try {
          const currentUser = await getCurrentUser()
          if (requestVersion.current === version) {
            setUser(currentUser)
            setErrorMessage(null)
            setStatus("authenticated")
          }
        } catch (recheckError) {
          if (requestVersion.current === version) {
            setUser(null)
            if (
              recheckError instanceof AuthApiError &&
              recheckError.status === 401
            ) {
              setErrorMessage(null)
              setStatus("unauthenticated")
            } else {
              setErrorMessage(safeErrorMessage(recheckError))
              setStatus("error")
            }
          }
        }
        throw error
      }

      if (requestVersion.current === version) {
        setUser(authenticatedUser)
        setErrorMessage(null)
        setStatus("authenticated")
      }

      return authenticatedUser
    },
    [],
  )

  const logout = useCallback(async () => {
    const version = ++requestVersion.current
    try {
      await logoutRequest()
    } catch (error) {
      if (error instanceof AuthApiError) {
        try {
          const currentUser = await getCurrentUser()
          if (requestVersion.current === version) {
            setUser(currentUser)
            setErrorMessage(null)
            setStatus("authenticated")
          }
        } catch (recheckError) {
          if (
            requestVersion.current === version &&
            recheckError instanceof AuthApiError &&
            recheckError.status === 401
          ) {
            setUser(null)
            setErrorMessage(null)
            setStatus("unauthenticated")
          }
        }
      }
      throw error
    }

    if (requestVersion.current === version) {
      setUser(null)
      setErrorMessage(null)
      setStatus("unauthenticated")
    }
  }, [])

  const value = useMemo<AuthContextValue>(
    () => ({
      status,
      user,
      errorMessage,
      login,
      logout,
      refreshCurrentUser,
    }),
    [errorMessage, login, logout, refreshCurrentUser, status, user],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext)

  if (!context) {
    throw new Error("useAuth must be used inside AuthProvider")
  }

  return context
}
