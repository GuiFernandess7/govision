import { API_BASE, authFetch, getEmail, logout, parseJsonSafe, requireAuth } from "./api.js";

requireAuth();

// ── Constants ────────────────────────────────────────────────

const MAX_FILE_SIZE = 5 * 1024 * 1024;
const MAX_UPLOAD_CONCURRENCY = 3;
const POLL_INTERVAL_MS = 3_000;
const TRUNCATE_ID = 22;
const TRUNCATE_FILE = 22;
const TERMINAL_STATUSES = new Set(["completed", "failed"]);
const BOX_COLORS = [
    "#ef4444", "#3b82f6", "#22c55e", "#f59e0b", "#a855f7",
    "#ec4899", "#06b6d4", "#f97316", "#14b8a6", "#8b5cf6",
];

// ── DOM references ───────────────────────────────────────────

const $ = (/** @type {string} */ id) => document.getElementById(id);

const dom = Object.freeze({
    dropZone:    $("dropZone"),
    fileInput:   $("fileInput"),
    filePreview: $("filePreview"),
    fileList:    $("fileList"),
    uploadBtn:   $("uploadBtn"),
    jobsBody:    $("jobsBody"),
    jobCount:    $("jobCount"),
    emptyState:  $("emptyState"),
    userEmail:   $("userEmail"),
    logoutBtn:   $("logoutBtn"),
    modal:       $("imageModal"),
    canvas:      $("modalCanvas"),
    modalClose:  $("modalClose"),
    downloadBtn: $("downloadBtn"),
});

// ── Application state ────────────────────────────────────────

/** @type {File[]} */
let selectedFiles = [];

/** @type {number | null} */
let pollTimerId = null;

/** @type {string | null} */
let currentModalJobId = null;

/** @type {Map<string, JobEntry>} */
const jobs = new Map();

/**
 * @typedef {{
 *   id: string,
 *   fileName: string,
 *   status: string,
 *   imageUrl: string | null,
 *   predictions: Array<Record<string, unknown>>,
 *   error: string | null,
 *   downloaded: boolean,
 *   createdAt: number,
 *   isTemp: boolean,
 * }} JobEntry
 */

// ── Initialisation ───────────────────────────────────────────

dom.userEmail.textContent = getEmail();

dom.logoutBtn.addEventListener("click", logout);
dom.dropZone.addEventListener("click", () => dom.fileInput.click());
dom.dropZone.addEventListener("keydown", (e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); dom.fileInput.click(); } });
dom.dropZone.addEventListener("dragover", onDragOver);
dom.dropZone.addEventListener("dragleave", onDragLeave);
dom.dropZone.addEventListener("drop", onDrop);
dom.fileInput.addEventListener("change", onFileInputChange);
dom.uploadBtn.addEventListener("click", onUploadAll);
dom.jobsBody.addEventListener("click", onJobsTableClick);
dom.modalClose.addEventListener("click", closeModal);
dom.modal.addEventListener("click", (e) => { if (e.target === dom.modal) closeModal(); });
dom.downloadBtn.addEventListener("click", onModalDownload);
document.addEventListener("keydown", (e) => { if (e.key === "Escape" && !dom.modal.hidden) closeModal(); });
window.addEventListener("beforeunload", stopPolling);

// ── File selection ───────────────────────────────────────────

function onDragOver(/** @type {DragEvent} */ event) {
    event.preventDefault();
    dom.dropZone.classList.add("dragover");
}

function onDragLeave() {
    dom.dropZone.classList.remove("dragover");
}

function onDrop(/** @type {DragEvent} */ event) {
    event.preventDefault();
    dom.dropZone.classList.remove("dragover");
    addFiles(event.dataTransfer?.files);
}

function onFileInputChange() {
    addFiles(dom.fileInput.files);
    dom.fileInput.value = "";
}

/** @param {FileList | null | undefined} files */
function addFiles(files) {
    if (!files) return;

    for (const file of files) {
        const valid = /^image\/(jpeg|png|gif)$/i.test(file.type) && file.size <= MAX_FILE_SIZE;
        if (!valid) continue;

        const duplicate = selectedFiles.some((f) => f.name === file.name && f.size === file.size);
        if (!duplicate) selectedFiles.push(file);
    }

    renderFileList();
}

function renderFileList() {
    dom.filePreview.hidden = selectedFiles.length === 0;
    dom.fileList.innerHTML = "";

    const fragment = document.createDocumentFragment();

    selectedFiles.forEach((file, index) => {
        const chip = document.createElement("div");
        chip.className = "file-chip";

        const name = document.createElement("span");
        name.textContent = truncate(file.name, TRUNCATE_FILE);

        const btn = document.createElement("button");
        btn.type = "button";
        btn.title = "Remove";
        btn.setAttribute("aria-label", `Remove ${file.name}`);
        btn.textContent = "\u00d7";
        btn.addEventListener("click", () => { selectedFiles.splice(index, 1); renderFileList(); });

        chip.append(name, btn);
        fragment.appendChild(chip);
    });

    dom.fileList.appendChild(fragment);
}

// ── Upload ───────────────────────────────────────────────────

async function onUploadAll() {
    if (selectedFiles.length === 0) return;

    setUploadLoading(true);
    const batch = [...selectedFiles];
    selectedFiles = [];
    renderFileList();

    await runWithConcurrency(batch, MAX_UPLOAD_CONCURRENCY, uploadSingleFile);
    setUploadLoading(false);
}

function setUploadLoading(loading) {
    dom.uploadBtn.disabled = loading;
    dom.uploadBtn.textContent = loading ? "Uploading..." : "Upload All";
}

/**
 * Runs `worker` on each item with at most `concurrency` in-flight.
 * @template T
 * @param {T[]} items
 * @param {number} concurrency
 * @param {(item: T) => Promise<void>} worker
 */
async function runWithConcurrency(items, concurrency, worker) {
    const queue = [...items];
    const lanes = Array.from({ length: Math.min(concurrency, queue.length) }, () =>
        (async () => {
            while (queue.length > 0) {
                const item = queue.shift();
                if (item) {
                    try { await worker(item); } catch { /* handled inside worker */ }
                }
            }
        })()
    );
    await Promise.allSettled(lanes);
}

/** @param {File} file */
async function uploadSingleFile(file) {
    const tempId = createTempJob(file.name);

    try {
        const formData = new FormData();
        formData.append("file", file);

        const response = await authFetch(`${API_BASE}/image/upload`, {
            method: "POST",
            body: formData,
        });

        const data = await parseJsonSafe(response);
        jobs.delete(tempId);

        if (!response.ok || !data?.job_id) {
            setJob(tempId, { fileName: file.name, status: "failed", error: data?.message || "Upload failed", isTemp: true });
        } else {
            setJob(/** @type {string} */ (data.job_id), {
                fileName: file.name,
                status: /** @type {string} */ (data.status) || "queued",
                isTemp: false,
            });
            ensurePolling();
        }
    } catch {
        jobs.delete(tempId);
        setJob(tempId, { fileName: file.name, status: "failed", error: "Network error", isTemp: true });
    }

    renderJobs();
}

// ── Job state management ─────────────────────────────────────

/** @returns {JobEntry} */
function defaultJob(id) {
    return {
        id,
        fileName: "",
        status: "queued",
        imageUrl: null,
        predictions: [],
        error: null,
        downloaded: false,
        createdAt: Date.now(),
        isTemp: false,
    };
}

/**
 * Creates a temporary "uploading" placeholder and renders the table.
 * @returns {string} the temporary id
 */
function createTempJob(fileName) {
    const id = `temp-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
    jobs.set(id, { ...defaultJob(id), fileName, status: "uploading", isTemp: true });
    renderJobs();
    return id;
}

/** Merges `partial` into the existing job (or creates a new one). */
function setJob(id, partial) {
    const current = jobs.get(id) || defaultJob(id);
    jobs.set(id, { ...current, ...partial, id });
}

// ── Polling ──────────────────────────────────────────────────

function ensurePolling() {
    if (pollTimerId !== null) return;
    pollTimerId = window.setInterval(pollPendingJobs, POLL_INTERVAL_MS);
    pollPendingJobs();
}

function stopPolling() {
    if (pollTimerId !== null) {
        clearInterval(pollTimerId);
        pollTimerId = null;
    }
}

async function pollPendingJobs() {
    const pendingIds = [];
    for (const [id, job] of jobs) {
        if (!job.isTemp && !TERMINAL_STATUSES.has(job.status)) {
            pendingIds.push(id);
        }
    }

    if (pendingIds.length === 0) {
        stopPolling();
        return;
    }

    await Promise.allSettled(pendingIds.map(fetchJobStatus));
    renderJobs();
}

/** @param {string} jobId */
async function fetchJobStatus(jobId) {
    try {
        const response = await authFetch(`${API_BASE}/jobs/${jobId}`);
        if (!response.ok) return;

        const data = await parseJsonSafe(response);
        if (!data) return;

        const job = jobs.get(jobId);
        if (!job) return;

        const wasPending = job.status !== "completed";

        setJob(jobId, {
            status: /** @type {string} */ (data.status) || job.status,
            imageUrl: /** @type {string} */ (data.image_url) || job.imageUrl,
            predictions: Array.isArray(data.predictions) ? data.predictions : job.predictions,
        });

        const updated = jobs.get(jobId);
        if (wasPending && updated?.status === "completed" && !updated.downloaded) {
            await autoDownload(jobId);
            setJob(jobId, { downloaded: true });
        }
    } catch {
        // will retry on next poll cycle
    }
}

// ── Jobs table rendering ─────────────────────────────────────

function renderJobs() {
    const entries = Array.from(jobs.values()).sort((a, b) => b.createdAt - a.createdAt);

    dom.jobCount.textContent = String(entries.length);
    dom.emptyState.hidden = entries.length > 0;
    dom.jobsBody.innerHTML = "";

    if (entries.length === 0) return;

    const fragment = document.createDocumentFragment();

    for (const job of entries) {
        const tr = document.createElement("tr");
        tr.append(
            createCell(createMonoSpan(job.isTemp ? "..." : job.id, TRUNCATE_ID)),
            createCell(createTruncatedSpan(job.fileName, TRUNCATE_FILE, "file-name")),
            createCell(createStatusBadge(job.status)),
            createTextCell(job.status === "completed" ? String(job.predictions.length) : "\u2014"),
            createResultCell(job),
        );
        fragment.appendChild(tr);
    }

    dom.jobsBody.appendChild(fragment);
}

/** @returns {HTMLTableCellElement} */
function createCell(child) {
    const td = document.createElement("td");
    td.appendChild(child);
    return td;
}

/** @returns {HTMLTableCellElement} */
function createTextCell(text) {
    const td = document.createElement("td");
    td.textContent = text;
    return td;
}

function createMonoSpan(text, maxLen) {
    const span = document.createElement("span");
    span.className = "job-id";
    span.title = text;
    span.textContent = truncate(text, maxLen);
    return span;
}

function createTruncatedSpan(text, maxLen, className) {
    const span = document.createElement("span");
    span.className = className;
    span.title = text;
    span.textContent = truncate(text, maxLen);
    return span;
}

/** @returns {HTMLSpanElement} */
function createStatusBadge(status) {
    const STATUS_CLASSES = {
        uploading: "status-uploading",
        queued:    "status-queued",
        pending:   "status-pending",
        completed: "status-completed",
        failed:    "status-failed",
    };

    const badge = document.createElement("span");
    badge.className = `status ${STATUS_CLASSES[status] || "status-queued"}`;

    if (!TERMINAL_STATUSES.has(status)) {
        const dot = document.createElement("span");
        dot.className = "spinner-dot";
        badge.appendChild(dot);
    }

    badge.append(status);
    return badge;
}

/** @returns {HTMLTableCellElement} */
function createResultCell(job) {
    const td = document.createElement("td");

    if (job.status === "completed") {
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "btn btn-sm btn-primary view-btn";
        btn.dataset.id = job.id;
        btn.textContent = "View";
        td.appendChild(btn);
    } else if (job.status === "failed") {
        const label = document.createElement("span");
        label.textContent = job.error || "Error";
        td.appendChild(label);
    } else {
        td.textContent = "\u2014";
    }

    return td;
}

/** @param {MouseEvent} event */
function onJobsTableClick(event) {
    const button = /** @type {HTMLElement} */ (event.target).closest(".view-btn");
    if (button?.dataset.id) {
        void showAnnotated(button.dataset.id);
    }
}

// ── Canvas / bounding box drawing ────────────────────────────

/**
 * Draws bounding boxes over the source image on a canvas.
 * Predictions use **center-based** coordinates.
 * @param {HTMLCanvasElement} canvas
 * @param {HTMLImageElement} image
 * @param {Array<Record<string, unknown>>} predictions
 */
function drawBoundingBoxes(canvas, image, predictions) {
    const ctx = canvas.getContext("2d");
    canvas.width = image.naturalWidth;
    canvas.height = image.naturalHeight;
    ctx.drawImage(image, 0, 0);

    const lineWidth = Math.max(2, Math.round(image.naturalWidth / 300));
    const fontSize = Math.max(12, Math.round(image.naturalWidth / 40));
    ctx.lineWidth = lineWidth;
    ctx.font = `bold ${fontSize}px sans-serif`;

    for (const p of predictions) {
        const w = Number(p.width) || 0;
        const h = Number(p.height) || 0;
        if (w <= 0 || h <= 0) continue;

        const cx = Number(p.x) || 0;
        const cy = Number(p.y) || 0;
        const x = cx - w / 2;
        const y = cy - h / 2;

        const classId = Number(p.class_id) || 0;
        const color = BOX_COLORS[Math.abs(classId) % BOX_COLORS.length];
        const confidence = Number(p.confidence) || 0;
        const label = `${p.class || "object"} ${(confidence * 100).toFixed(0)}%`;

        // box
        ctx.strokeStyle = color;
        ctx.strokeRect(x, y, w, h);

        // label background
        const metrics = ctx.measureText(label);
        const labelH = fontSize + 6;
        const labelW = metrics.width + 8;
        const labelY = Math.max(0, y - labelH);

        ctx.fillStyle = color;
        ctx.fillRect(x, labelY, labelW, labelH);

        // label text
        ctx.fillStyle = "#fff";
        ctx.fillText(label, x + 4, labelY + labelH - 4);
    }
}

/**
 * @param {string} url
 * @returns {Promise<HTMLImageElement>}
 */
function loadImage(url) {
    return new Promise((resolve, reject) => {
        const img = new Image();
        img.crossOrigin = "anonymous";
        img.onload = () => resolve(img);
        img.onerror = () => reject(new Error(`Failed to load: ${url}`));
        img.src = url;
    });
}

/** Triggers a PNG download of the canvas contents. */
function downloadCanvas(canvas, jobId) {
    const a = document.createElement("a");
    a.download = `govision-${jobId}.png`;
    a.href = canvas.toDataURL("image/png");
    a.click();
}

/** Renders bounding boxes off-screen and auto-downloads the result. */
async function autoDownload(jobId) {
    const job = jobs.get(jobId);
    if (!job?.imageUrl) return;

    try {
        const image = await loadImage(job.imageUrl);
        const offscreen = document.createElement("canvas");
        drawBoundingBoxes(offscreen, image, job.predictions);
        downloadCanvas(offscreen, jobId);
    } catch {
        // image may be cross-origin blocked; silently skip
    }
}

// ── Modal ────────────────────────────────────────────────────

async function showAnnotated(jobId) {
    const job = jobs.get(jobId);
    if (!job?.imageUrl) return;

    try {
        const image = await loadImage(job.imageUrl);
        drawBoundingBoxes(/** @type {HTMLCanvasElement} */ (dom.canvas), image, job.predictions);
        currentModalJobId = jobId;
        dom.modal.hidden = false;
    } catch {
        // image load failed; no-op
    }
}

function closeModal() {
    dom.modal.hidden = true;
    currentModalJobId = null;
}

function onModalDownload() {
    if (!currentModalJobId) return;
    downloadCanvas(/** @type {HTMLCanvasElement} */ (dom.canvas), currentModalJobId);
}

// ── Utilities ────────────────────────────────────────────────

/** @param {string} value @param {number} max */
function truncate(value, max) {
    return value.length > max ? `${value.slice(0, max - 1)}\u2026` : value;
}
