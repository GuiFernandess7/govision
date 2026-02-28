import { postJson } from "./api.js";

const form = document.getElementById("registerForm");
const errorMsg = document.getElementById("errorMsg");
const successMsg = document.getElementById("successMsg");
const submitBtn = document.getElementById("submitBtn");

const LABEL_DEFAULT = "Create Account";
const LABEL_LOADING = "Creating...";
const REDIRECT_DELAY_MS = 1500;

function setLoading(loading) {
    submitBtn.disabled = loading;
    submitBtn.textContent = loading ? LABEL_LOADING : LABEL_DEFAULT;
}

function showError(message) {
    successMsg.hidden = true;
    errorMsg.textContent = message;
    errorMsg.hidden = false;
}

function showSuccess(message) {
    errorMsg.hidden = true;
    successMsg.textContent = message;
    successMsg.hidden = false;
}

function hideMessages() {
    errorMsg.hidden = true;
    successMsg.hidden = true;
}

function validate(email, password, confirmPassword) {
    if (!email || !password) {
        return "Email and password are required.";
    }
    if (password !== confirmPassword) {
        return "Passwords do not match.";
    }
    return null;
}

form.addEventListener("submit", async (event) => {
    event.preventDefault();
    hideMessages();
    setLoading(true);

    const email = form.email.value.trim();
    const password = form.password.value;
    const confirmPassword = form.confirmPassword.value;

    const validationError = validate(email, password, confirmPassword);
    if (validationError) {
        showError(validationError);
        setLoading(false);
        return;
    }

    try {
        const { response, data } = await postJson("/auth/register", { email, password });

        if (!response.ok) {
            showError(data?.message || "Registration failed.");
            return;
        }

        showSuccess("Account created! Redirecting to login...");
        form.reset();
        setTimeout(() => { window.location.href = "index.html"; }, REDIRECT_DELAY_MS);
    } catch {
        showError("Network error. Please try again.");
    } finally {
        setLoading(false);
    }
});
