// This file contains JS code for the browser-side zip grouping feature.
// All files named admin_*.js will be merged together and minimised by calling
// go generate ./...
//
// Feature: when the "Zip Archive" upload option is enabled (default), files
// added to the dropzone are collected into a batch, compressed into a single
// zip archive in the browser using JSZip, and then handed back to the dropzone
// as one regular file upload. The original file structure (relative paths of
// dropped folders) is preserved inside the archive.


// Files added within this window (ms) are grouped into a single archive. This
// lets a multi-file selection or a folder drop arrive as one archive instead of
// starting a new zip for every individual file event.
const ZIP_BATCH_DEBOUNCE_MS = 400;

// Marker property set on the generated zip File so that the addedfile handler
// does not try to re-compress it (which would cause an infinite loop).
const ZIP_GENERATED_FLAG = "isGokapiZipArchive";

// Buffer of files waiting to be compressed and the timer that flushes them.
var zipPendingFiles = [];
var zipBatchTimer = null;

// Incrementing id so that each in-progress archive gets its own status row.
var zipStatusCounter = 0;


/**
 * isZipGroupingEnabled returns true if the user has enabled the zip option.
 * The checkbox is disabled (and therefore never checked) when end-to-end
 * encryption is active, so E2E uploads always take the normal per-file path.
 */
function isZipGroupingEnabled() {
    let checkbox = document.getElementById("enableZip");
    return checkbox != null && checkbox.checked && !checkbox.disabled;
}


/**
 * initZipDefaults restores the zip checkbox state and name template from a
 * previous session (if any) and wires up handlers to persist changes. It is
 * a no-op when the fields are absent (e.g. E2E view without the controls).
 * Called once after the page loads.
 */
function initZipDefaults() {
    let checkbox = document.getElementById("enableZip");
    let templateInput = document.getElementById("zipNameTemplate");

    if (checkbox != null && !checkbox.disabled) {
        // getLocalStorageWithDefault is defined in admin_ui_upload.js.
        let storedEnabled = getLocalStorageWithDefault("defaultZipEnabled", "true");
        checkbox.checked = storedEnabled === "true";
        checkbox.addEventListener("change", () => {
            localStorage.setItem("defaultZipEnabled", checkbox.checked);
        });
    }

    if (templateInput != null) {
        let storedTemplate = localStorage.getItem("defaultZipNameTemplate");
        if (storedTemplate !== null && storedTemplate !== "") {
            templateInput.value = storedTemplate;
        }
        templateInput.addEventListener("change", () => {
            localStorage.setItem("defaultZipNameTemplate", templateInput.value);
        });
    }
}


/**
 * handleZipAddedFile is called from the dropzone "addedfile" handler.
 *
 * Returns true if this file has been taken over by the zip feature (i.e. it was
 * buffered for compression and must not follow the normal upload path).
 * Returns false if the file should be uploaded normally (zip disabled, or this
 * is the already-generated archive itself).
 *
 * @param {File} file
 * @returns {boolean}
 */
function handleZipAddedFile(file) {
    // The generated archive itself must upload normally.
    if (file[ZIP_GENERATED_FLAG] === true) {
        return false;
    }
    if (!isZipGroupingEnabled()) {
        return false;
    }

    // Remove the individual file from the dropzone queue so it is not uploaded
    // on its own. Its bytes are still fully accessible via the File reference we
    // keep here.
    try {
        dropzoneObject.removeFile(file);
    } catch (e) {
        console.warn("Could not remove buffered file from dropzone:", e);
    }

    zipPendingFiles.push(file);

    // (Re)start the debounce timer. Once it fires, the whole batch is zipped.
    if (zipBatchTimer !== null) {
        clearTimeout(zipBatchTimer);
    }
    zipBatchTimer = setTimeout(flushZipBatch, ZIP_BATCH_DEBOUNCE_MS);

    return true;
}


/**
 * flushZipBatch compresses all currently buffered files into a single archive
 * and adds the resulting file back to the dropzone as a normal upload.
 */
function flushZipBatch() {
    zipBatchTimer = null;

    if (zipPendingFiles.length === 0) {
        return;
    }

    // Take ownership of the current batch and reset the buffer so that files
    // added while we compress start a fresh batch.
    let batch = zipPendingFiles;
    zipPendingFiles = [];

    // A single loose file (not part of a dropped folder) is never zipped -
    // wrapping one file in an archive adds no value. It is uploaded as-is.
    // Note: a single file that came from inside a folder still gets zipped,
    // so that the folder upload rule (folder name + date) is preserved.
    if (batch.length === 1 && getTopLevelFolder(batch) === null) {
        addFilesWithoutZip(batch);
        return;
    }

    if (typeof JSZip === "undefined") {
        // Safety net: if the library failed to load, fall back to uploading the
        // files individually rather than losing them.
        console.error("JSZip is not available - uploading files without compression.");
        addFilesWithoutZip(batch);
        return;
    }

    let statusId = "zip-" + (zipStatusCounter++);
    let archiveName = generateZipFilename(batch);
    addZipStatus(statusId, archiveName);

    let zip = new JSZip();
    for (let file of batch) {
        // Preserve the original structure: dropzone sets fullPath for files that
        // came from a dropped directory. Fall back to the plain file name.
        let entryName = getRelativePathForFile(file);
        zip.file(entryName, file, {
            date: file.lastModified ? new Date(file.lastModified) : new Date(),
        });
    }

    zip.generateAsync({
            type: "blob",
            compression: "DEFLATE",
            compressionOptions: {
                level: 6,
            },
        }, (metadata) => {
            updateZipStatus(statusId, "Compressing... " + Math.round(metadata.percent) + "%", metadata.percent);
        })
        .then((blob) => {
            let archive = new File([blob], archiveName, {
                type: "application/zip",
                lastModified: Date.now(),
            });
            // Mark the archive so the addedfile handler lets it through.
            archive[ZIP_GENERATED_FLAG] = true;

            updateZipStatus(statusId, "Compressed (" + formatZipBytes(blob.size) + ") - queued for upload", 100);
            setTimeout(() => removeZipStatus(statusId), 1500);

            // Hand the finished archive to the dropzone as a normal upload.
            dropzoneObject.addFile(archive);
        })
        .catch((err) => {
            console.error("Zip compression failed:", err);
            setZipStatusError(statusId, "Compression failed: " + (err && err.message ? err.message : err));
        });
}


/**
 * addFilesWithoutZip is the fallback path used when JSZip is unavailable. Each
 * file is re-added to the dropzone but flagged as generated so it is not
 * re-intercepted, causing it to upload individually.
 * @param {File[]} files
 */
function addFilesWithoutZip(files) {
    for (let file of files) {
        file[ZIP_GENERATED_FLAG] = true;
        dropzoneObject.addFile(file);
    }
}


/**
 * getRelativePathForFile returns the path used for the file inside the archive,
 * preserving any directory structure from a dropped folder.
 * @param {File} file
 * @returns {string}
 */
function getRelativePathForFile(file) {
    // Dropzone stores the relative path (for directory drops) on fullPath.
    if (file.fullPath && typeof file.fullPath === "string" && file.fullPath !== "") {
        return file.fullPath.replace(/^\/+/, "");
    }
    // Some browsers expose webkitRelativePath for <input webkitdirectory>.
    if (file.webkitRelativePath && file.webkitRelativePath !== "") {
        return file.webkitRelativePath;
    }
    return file.name;
}


/**
 * getTopLevelFolder inspects the batch and, if every file lives under one single
 * top-level directory, returns that directory's name. Otherwise returns null
 * (mixed files, or files with no directory structure).
 *
 * This is used so that dropping a single folder produces an archive named after
 * that folder plus the date/time, e.g. "myfolder-20260807-112025.zip".
 *
 * @param {File[]} batch
 * @returns {string|null}
 */
function getTopLevelFolder(batch) {
    let topFolder = null;
    for (let file of batch) {
        let relPath = getRelativePathForFile(file);
        // A file that carries directory structure contains a slash.
        let slashIndex = relPath.indexOf("/");
        if (slashIndex === -1) {
            // A loose file with no folder -> this is not a single-folder upload.
            return null;
        }
        let firstSegment = relPath.substring(0, slashIndex);
        if (firstSegment === "") {
            return null;
        }
        if (topFolder === null) {
            topFolder = firstSegment;
        } else if (topFolder !== firstSegment) {
            // Files span more than one top-level folder.
            return null;
        }
    }
    return topFolder;
}


/**
 * strftime renders a small subset of strftime-style tokens against a Date.
 * Supported tokens: %Y %y %m %d %H %M %S %j %% .
 * Unknown tokens are left as-is (minus the leading %).
 *
 * @param {string} pattern
 * @param {Date} date
 * @returns {string}
 */
function strftime(pattern, date) {
    let pad = (n, len) => String(n).padStart(len || 2, "0");
    let dayOfYear = () => {
        let start = new Date(date.getFullYear(), 0, 0);
        let diff = date - start;
        return Math.floor(diff / 86400000);
    };
    return pattern.replace(/%([A-Za-z%])/g, (match, token) => {
        switch (token) {
            case "Y": return String(date.getFullYear());
            case "y": return pad(date.getFullYear() % 100);
            case "m": return pad(date.getMonth() + 1);
            case "d": return pad(date.getDate());
            case "H": return pad(date.getHours());
            case "M": return pad(date.getMinutes());
            case "S": return pad(date.getSeconds());
            case "j": return pad(dayOfYear(), 3);
            case "%": return "%";
            default: return token;
        }
    });
}


/**
 * sanitizeArchiveName strips characters that are unsafe or awkward in a
 * download filename and trims surrounding whitespace/dots.
 * @param {string} name
 * @returns {string}
 */
function sanitizeArchiveName(name) {
    // Remove path separators and characters disallowed on common filesystems.
    let cleaned = name.replace(/[\/\\:*?"<>|]+/g, "_").trim();
    // Collapse whitespace and strip leading/trailing dots.
    cleaned = cleaned.replace(/\s+/g, " ").replace(/^\.+|\.+$/g, "");
    return cleaned;
}


/**
 * generateZipFilename builds the archive name for a batch.
 *
 * Rules:
 *  - If the batch is a single top-level folder, the name is
 *    "<foldername>-<YYYYMMDD-HHMMSS>.zip" (ignoring the template field).
 *  - Otherwise the user's name template is rendered with strftime tokens.
 *    An empty template falls back to the default prefix + timestamp.
 *  - A ".zip" extension is always ensured.
 *
 * @param {File[]} [batch]
 * @returns {string}
 */
function generateZipFilename(batch) {
    let now = new Date();
    let dateStamp = strftime("%Y%m%d-%H%M%S", now);

    // Single-folder upload: name after the folder + date-time.
    if (Array.isArray(batch) && batch.length > 0) {
        let folder = getTopLevelFolder(batch);
        if (folder !== null && folder !== "") {
            let name = sanitizeArchiveName(folder) + "-" + dateStamp;
            return ensureZipExtension(name);
        }
    }

    // Otherwise use the user-provided template. When the field is left empty,
    // fall back to the server-configured default (GOKAPI_DEFAULT_ZIP_NAME,
    // injected as defaultZipNameTemplate) and finally to a hardcoded default.
    let templateInput = document.getElementById("zipNameTemplate");
    let template = templateInput != null ? templateInput.value.trim() : "";
    if (template === "") {
        if (typeof defaultZipNameTemplate === "string" && defaultZipNameTemplate.trim() !== "") {
            template = defaultZipNameTemplate.trim();
        } else {
            template = "gokapi-upload-%Y%m%d-%H%M%S";
        }
    }

    let rendered = sanitizeArchiveName(strftime(template, now));
    if (rendered === "") {
        rendered = "gokapi-upload-" + dateStamp;
    }
    return ensureZipExtension(rendered);
}


/**
 * ensureZipExtension appends ".zip" unless the name already ends with it.
 * @param {string} name
 * @returns {string}
 */
function ensureZipExtension(name) {
    if (/\.zip$/i.test(name)) {
        return name;
    }
    return name + ".zip";
}


/**
 * formatZipBytes returns a human-readable size string.
 * @param {number} bytes
 * @returns {string}
 */
function formatZipBytes(bytes) {
    if (bytes < 1024) {
        return bytes + " B";
    }
    let units = ["KB", "MB", "GB", "TB"];
    let value = bytes / 1024;
    let unitIndex = 0;
    while (value >= 1024 && unitIndex < units.length - 1) {
        value /= 1024;
        unitIndex++;
    }
    return (Math.round(value * 10) / 10) + " " + units[unitIndex];
}


// ── Compression status UI ────────────────────────────────────────────────
// Reuses the same visual containers as the upload progress rows so the
// compression phase looks consistent with the rest of the upload UI.

function addZipStatus(statusId, archiveName) {
    const container = document.createElement("div");
    container.setAttribute("id", `us-container-${statusId}`);
    container.classList.add("us-container");

    const filenameDiv = document.createElement("div");
    filenameDiv.classList.add("filename");
    filenameDiv.textContent = archiveName;
    container.appendChild(filenameDiv);

    const progressContainerDiv = document.createElement("div");
    progressContainerDiv.classList.add("upload-progress-container");

    const progressBarDiv = document.createElement("div");
    progressBarDiv.classList.add("upload-progress-bar");

    const progressBarProgressDiv = document.createElement("div");
    progressBarProgressDiv.setAttribute("id", `us-progressbar-${statusId}`);
    progressBarProgressDiv.classList.add("upload-progress-bar-progress");
    progressBarProgressDiv.style.width = "0%";
    progressBarDiv.appendChild(progressBarProgressDiv);

    const progressInfoDiv = document.createElement("div");
    progressInfoDiv.setAttribute("id", `us-progress-info-${statusId}`);
    progressInfoDiv.classList.add("upload-progress-info");
    progressInfoDiv.textContent = "Preparing archive...";

    progressContainerDiv.appendChild(progressBarDiv);
    progressContainerDiv.appendChild(progressInfoDiv);
    container.appendChild(progressContainerDiv);

    const uploadstatusContainer = document.getElementById("uploadstatus");
    uploadstatusContainer.appendChild(container);
    uploadstatusContainer.style.visibility = "visible";
}

function updateZipStatus(statusId, text, percent) {
    let bar = document.getElementById(`us-progressbar-${statusId}`);
    let info = document.getElementById(`us-progress-info-${statusId}`);
    if (bar != null && typeof percent === "number") {
        let rounded = Math.max(0, Math.min(100, Math.round(percent)));
        bar.style.width = rounded + "%";
    }
    if (info != null) {
        info.innerText = text;
    }
}

function setZipStatusError(statusId, message) {
    let bar = document.getElementById(`us-progressbar-${statusId}`);
    let info = document.getElementById(`us-progress-info-${statusId}`);
    if (bar != null) {
        bar.style.width = "100%";
        bar.style.backgroundColor = "red";
    }
    if (info != null) {
        info.innerText = message;
        info.classList.add("uploaderror");
    }
}

function removeZipStatus(statusId) {
    const container = document.getElementById(`us-container-${statusId}`);
    if (container != null) {
        container.remove();
    }
    if (document.querySelectorAll("#uploadstatus .us-container").length < 1) {
        document.getElementById("uploadstatus").style.visibility = "hidden";
    }
}
