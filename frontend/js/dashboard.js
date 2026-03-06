import { API_BASE, authFetch, getEmail, logout, parseJsonSafe, requireAuth } from "./api.js";

requireAuth();

// ── Constants ────────────────────────────────────────────────

const MAX_FILE_SIZE = 14 * 1024 * 1024;
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
});

// ── Application state ────────────────────────────────────────

/** @type {File[]} */
let selectedFiles = [];

/** @type {number | null} */
let pollTimerId = null;

/** @type {Set<string>} */
const expandedJobs = new Set();

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
        const isCompleted = job.status === "completed";
        const isExpanded = expandedJobs.has(job.id);

        // Main row
        const tr = document.createElement("tr");
        if (isCompleted) {
            tr.className = `job-row-clickable${isExpanded ? " expanded" : ""}`;
            tr.dataset.jobId = job.id;
            tr.title = "Click to view results";
        }
        tr.append(
            createCell(createMonoSpan(job.isTemp ? "..." : job.id, TRUNCATE_ID)),
            createCell(createTruncatedSpan(job.fileName, TRUNCATE_FILE, "file-name")),
            createCell(createStatusBadge(job.status)),
            createTextCell(isCompleted ? String(job.predictions.length) : "\u2014"),
        );
        fragment.appendChild(tr);

        // Detail row (only for completed jobs)
        if (isCompleted) {
            const detailTr = document.createElement("tr");
            detailTr.className = `job-detail-row${isExpanded ? " open" : ""}`;
            detailTr.id = `detail-${job.id}`;

            const detailTd = document.createElement("td");
            detailTd.colSpan = 4;

            if (isExpanded) {
                buildDetailContent(detailTd, job);
            }

            detailTr.appendChild(detailTd);
            fragment.appendChild(detailTr);
        }
    }

    dom.jobsBody.appendChild(fragment);
}

/** Builds the expanded detail panel inside the given td. */
function buildDetailContent(td, job) {
    const wrapper = document.createElement("div");
    wrapper.className = "job-detail-content";

    // Left side — image with bounding boxes
    const imageDiv = document.createElement("div");
    imageDiv.className = "job-detail-image";

    const canvas = document.createElement("canvas");
    imageDiv.appendChild(canvas);

    if (job.imageUrl) {
        loadImage(job.imageUrl)
            .then((img) => drawBoundingBoxes(canvas, img, job.predictions))
            .catch(() => {
                canvas.style.display = "none";
                const err = document.createElement("p");
                err.textContent = "Unable to load image.";
                err.style.color = "var(--text-muted)";
                imageDiv.appendChild(err);
            });
    }

    // Right side — stats
    const statsDiv = document.createElement("div");
    statsDiv.className = "job-detail-stats";

    // Total objects card
    const totalCard = document.createElement("div");
    totalCard.className = "stat-card";
    const totalValue = document.createElement("div");
    totalValue.className = "stat-value";
    totalValue.textContent = String(job.predictions.length);
    const totalLabel = document.createElement("div");
    totalLabel.className = "stat-label";
    totalLabel.textContent = "Objects Detected";
    totalCard.append(totalValue, totalLabel);
    statsDiv.appendChild(totalCard);

    // Class breakdown
    const classCounts = {};
    for (const p of job.predictions) {
        const cls = String(p.class || "object");
        classCounts[cls] = (classCounts[cls] || 0) + 1;
    }

    if (Object.keys(classCounts).length > 0) {
        const breakdownCard = document.createElement("div");
        breakdownCard.className = "stat-card";

        const breakdownLabel = document.createElement("div");
        breakdownLabel.className = "stat-label";
        breakdownLabel.style.marginBottom = "0.5rem";
        breakdownLabel.textContent = "By class";
        breakdownCard.appendChild(breakdownLabel);

        const ul = document.createElement("ul");
        ul.className = "class-list";

        const sortedClasses = Object.entries(classCounts).sort((a, b) => b[1] - a[1]);
        for (const [cls, count] of sortedClasses) {
            const li = document.createElement("li");

            const nameSpan = document.createElement("span");
            const dot = document.createElement("span");
            dot.className = "class-dot";
            const classId = job.predictions.find((p) => String(p.class || "object") === cls)?.class_id || 0;
            dot.style.backgroundColor = BOX_COLORS[Math.abs(Number(classId)) % BOX_COLORS.length];
            nameSpan.append(dot, cls);

            const countSpan = document.createElement("span");
            countSpan.className = "class-count";
            countSpan.textContent = String(count);

            li.append(nameSpan, countSpan);
            ul.appendChild(li);
        }

        breakdownCard.appendChild(ul);
        statsDiv.appendChild(breakdownCard);
    }

    // Download button
    const dlBtn = document.createElement("button");
    dlBtn.type = "button";
    dlBtn.className = "btn btn-sm btn-primary detail-download-btn";
    dlBtn.textContent = "Download Image";
    dlBtn.addEventListener("click", (e) => {
        e.stopPropagation();
        downloadCanvas(canvas, job.id);
    });
    statsDiv.appendChild(dlBtn);

    wrapper.append(imageDiv, statsDiv);
    td.appendChild(wrapper);
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

/** @param {MouseEvent} event */
function onJobsTableClick(event) {
    const row = /** @type {HTMLElement} */ (event.target).closest(".job-row-clickable");
    if (!row?.dataset.jobId) return;

    const jobId = row.dataset.jobId;
    const detailRow = document.getElementById(`detail-${jobId}`);
    if (!detailRow) return;

    const isOpen = expandedJobs.has(jobId);

    if (isOpen) {
        expandedJobs.delete(jobId);
        row.classList.remove("expanded");
        detailRow.classList.remove("open");
    } else {
        expandedJobs.add(jobId);
        row.classList.add("expanded");
        detailRow.classList.add("open");

        // Lazy-build detail content on first expand
        const td = detailRow.querySelector("td");
        if (td && td.children.length === 0) {
            const job = jobs.get(jobId);
            if (job) buildDetailContent(td, job);
        }
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

// ── Utilities ────────────────────────────────────────────────

/** @param {string} value @param {number} max */
function truncate(value, max) {
    return value.length > max ? `${value.slice(0, max - 1)}\u2026` : value;
}
