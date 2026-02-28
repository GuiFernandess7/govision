import { postJson, saveAuth } from "./api.js";

const form = document.getElementById("loginForm");
const errorMsg = document.getElementById("errorMsg");
const submitBtn = document.getElementById("submitBtn");

const LABEL_DEFAULT = "Sign In";
const LABEL_LOADING = "Signing in...";

function setLoading(loading) {
    submitBtn.disabled = loading;
    submitBtn.textContent = loading ? LABEL_LOADING : LABEL_DEFAULT;
}

function showError(message) {
    errorMsg.textContent = message;
    errorMsg.hidden = false;
}

form.addEventListener("submit", async (event) => {
    event.preventDefault();
    errorMsg.hidden = true;
    setLoading(true);

    const email = form.email.value.trim();
    const password = form.password.value;

    if (!email || !password) {
        showError("Email and password are required.");
        setLoading(false);
        return;
    }

    try {
        const { response, data } = await postJson("/auth/login", { email, password });

        if (!response.ok) {
            showError(data?.message || "Login failed.");
            return;
        }

        if (!data?.access_token || !data?.refresh_token) {
            showError("Invalid server response. Please try again.");
            return;
        }

        saveAuth(/** @type {any} */ (data), email);
        window.location.href = "dashboard.html";
    } catch {
        showError("Network error. Please try again.");
    } finally {
        setLoading(false);
    }
});
