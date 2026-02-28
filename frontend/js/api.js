export const API_BASE = "http://localhost:8080/v1";

const TOKEN_KEY = "govision_access_token";
const REFRESH_KEY = "govision_refresh_token";
const EMAIL_KEY = "govision_email";

// ── Storage helpers ──────────────────────────────────────────

function clearAuthStorage() {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(REFRESH_KEY);
    localStorage.removeItem(EMAIL_KEY);
}

/** @returns {string | null} */
export function getAccessToken() {
    return localStorage.getItem(TOKEN_KEY);
}

/** @returns {string} */
export function getEmail() {
    return localStorage.getItem(EMAIL_KEY) || "";
}

/**
 * Persists auth tokens (and optionally the email) to localStorage.
 * @param {{ access_token: string, refresh_token: string }} tokens
 * @param {string} [email]
 */
export function saveAuth(tokens, email) {
    localStorage.setItem(TOKEN_KEY, tokens.access_token);
    localStorage.setItem(REFRESH_KEY, tokens.refresh_token);
    if (email) {
        localStorage.setItem(EMAIL_KEY, email);
    }
}

// ── Navigation guards ────────────────────────────────────────

/** Clears stored auth data and redirects to login. */
export function logout() {
    clearAuthStorage();
    window.location.href = "index.html";
}

/** Redirects to login when no access token is stored. */
export function requireAuth() {
    if (!getAccessToken()) {
        window.location.href = "index.html";
    }
}

// ── HTTP helpers ─────────────────────────────────────────────

/**
 * Safely parses a Response body as JSON.
 * Returns `null` when content-type is not JSON or parsing fails.
 * @param {Response} response
 * @returns {Promise<Record<string, unknown> | null>}
 */
export async function parseJsonSafe(response) {
    try {
        const contentType = response.headers.get("content-type") || "";
        if (!contentType.includes("application/json")) {
            return null;
        }
        return await response.json();
    } catch {
        return null;
    }
}

/**
 * Sends a JSON POST to `path` (relative to API_BASE) and returns
 * `{ response, data }` where `data` is the parsed body (or null).
 * @param {string} path  — e.g. "/auth/login"
 * @param {Record<string, unknown>} body
 * @returns {Promise<{ response: Response, data: Record<string, unknown> | null }>}
 */
export async function postJson(path, body) {
    const response = await fetch(`${API_BASE}${path}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
    });
    const data = await parseJsonSafe(response);
    return { response, data };
}

// ── Token refresh ────────────────────────────────────────────

/** @returns {Promise<boolean>} */
async function refreshAccessToken() {
    const refreshToken = localStorage.getItem(REFRESH_KEY);
    if (!refreshToken) {
        return false;
    }

    try {
        const { response, data } = await postJson("/auth/refresh", {
            refresh_token: refreshToken,
        });

        if (!response.ok || !data?.access_token || !data?.refresh_token) {
            return false;
        }

        saveAuth(/** @type {any} */ (data));
        return true;
    } catch {
        return false;
    }
}

// ── Authenticated fetch ──────────────────────────────────────

/**
 * Like `fetch` but injects the Bearer token and retries once on 401
 * after refreshing credentials.
 * @param {string} url
 * @param {RequestInit} [options]
 * @returns {Promise<Response>}
 */
export async function authFetch(url, options = {}) {
    const token = getAccessToken();
    const headers = {
        ...options.headers,
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
    };

    let response = await fetch(url, { ...options, headers });

    if (response.status === 401) {
        const refreshed = await refreshAccessToken();
        if (refreshed) {
            headers.Authorization = `Bearer ${getAccessToken()}`;
            response = await fetch(url, { ...options, headers });
        } else {
            logout();
        }
    }

    return response;
}
