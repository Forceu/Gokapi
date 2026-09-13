function createUploadBox() {

    fileInput.addEventListener('change', () => {
        Array.from(fileInput.files).forEach(file => {

            if (file.size > MAX_FILE_SIZE) {
                document.getElementById('span-modal-error').innerText =
                    t("pu_file_too_large", file.name, formatSize(MAX_FILE_SIZE));
                errorModal.show();
                return;
            }
            const uuid = getUuid();

            const item = document.createElement('div');
            item.className = 'pu-file-item';
            item.dataset.uuid = uuid;

            const name = document.createElement('span');
            name.textContent = file.name;
            name.className = 'file-name';

            const progressText = document.createElement('span');
            progressText.className = 'upload-status';
            progressText.textContent = t("pu_status_ready");

            const progressBar = document.createElement('progress');
            progressBar.className = 'upload-progress';

            if (file.size == 0) {
                progressBar.max = 1;
            } else {
                progressBar.max = file.size;
            }
            progressBar.value = 0;

            const size = document.createElement('span');
            size.className = 'file-size';
            size.textContent = formatSize(file.size);

            const removeBtn = document.createElement('button');
            removeBtn.type = 'button';
            removeBtn.title = t("btn_remove");
            removeBtn.className = 'btn btn-sm btn-link text-light p-0';
            removeBtn.innerHTML = '<i class="bi bi-x-circle"></i>';
            removeBtn.onclick = async () => {
                filesMap.get(uuid).removed = true;
                filesMap.get(uuid).status = 'removed';
                const entry = filesMap.get(uuid);

                // If currently uploading, abort
                if (entry.controller) {
                    entry.controller.abort();
                }
                item.remove();
                updateUploadButtonState();

                if (entry.serverUuid) {
                    try {
                        await unreserve(entry.serverUuid);
                    } catch (e) {
                        console.error("Unreserve failed", e);
                    }
                }
            };

            item.append(name, progressText, progressBar, size, removeBtn);
            fileList.appendChild(item);

            filesMap.set(uuid, {
                uuid,
                file,
                removed: false,
                status: 'pending',
                controller: new AbortController(),
                lastSpeed: "",
                elements: {
                    progressBar,
                    progressText,
                    removeBtn,
                    item
                }
            });
            updateUploadButtonState();
        });
        // Allow re-selecting same files
        fileInput.value = '';
    });



    // --- Drag and Drop Functionality ---

    // Prevent default behaviors for drag events
    ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
        uploadBox.addEventListener(eventName, (e) => {
            e.preventDefault();
            e.stopPropagation();
        }, false);
    });

    // Highlight box when dragging over
    ['dragenter', 'dragover'].forEach(eventName => {
        uploadBox.addEventListener(eventName, () => uploadBox.classList.add('highlight'), false);
    });

    ['dragleave', 'drop'].forEach(eventName => {
        uploadBox.addEventListener(eventName, () => uploadBox.classList.remove('highlight'), false);
    });

    // Handle dropped files
    uploadBox.addEventListener('drop', (e) => {
        const dt = e.dataTransfer;
        const files = dt.files;
        handleFiles(files);
    });

    // --- Paste Functionality ---

    window.addEventListener('paste', (e) => {
        const items = e.clipboardData.items;
        const files = [];

        for (let i = 0; i < items.length; i++) {
            // Handle Files (Images, etc)
            if (items[i].kind === 'file') {
                files.push(items[i].getAsFile());
            }
            // Handle Text pastes (converts text to a .txt file)
            else if (items[i].kind === 'string' && items[i].type === 'text/plain') {
                items[i].getAsString((text) => {
                    const blob = new Blob([text], {
                        type: 'text/plain'
                    });
                    const file = new File([blob], "pasted-text.txt", {
                        type: 'text/plain'
                    });
                    handleFiles([file]);
                });
            }
        }

        if (files.length > 0) {
            handleFiles(files);
        }
    });

}


function setUnload() {
    // Confirm before closing tab
    window.addEventListener('beforeunload', (e) => {
        const uploading = Array.from(filesMap.values()).some(f => !f.removed);
        if (uploading) {
            e.preventDefault();
            e.returnValue = '';
        }
    });

    // Attempt unreserve on actual exit
    window.addEventListener('unload', () => {
        for (const entry of filesMap.values()) {
            if (!entry.removed && entry.serverUuid) {
                unreserve(entry.serverUuid);
            }
        }
    });
}

function handleFiles(files) {
    const dataTransfer = new DataTransfer();
    Array.from(files).forEach(file => dataTransfer.items.add(file));
    fileInput.files = dataTransfer.files;
    fileInput.dispatchEvent(new Event('change'));
}

function updateUploadButtonState() {
    const btn = document.getElementById("uploadbutton");
    const pendingFiles = Array.from(filesMap.values()).filter(entry =>
        !entry.removed && entry.status === 'pending'
    );

    btn.disabled = isUploadInProgress || pendingFiles.length === 0;
}


function showModal(modalCode) {
    let message = "";
    switch (modalCode) {

        case "alluploaded":
            new bootstrap.Modal(document.getElementById('allUploadedModal'), {
                keyboard: false,
                backdrop: "static"
            }).show();
            return;

        case "maxfiles":
            if (maxFilesRemaining == 1) {
                message = t("pu_too_many_files_single");
            } else {
                message = t("pu_too_many_files", maxFilesRemaining);
            }
            break;

        case "maxfilesdynamic":
            message = t("pu_max_files_dynamic");
            break;

        case "expired":
            message = t("pu_request_expired");
            break;
    }
    document.getElementById('span-modal-error').innerText = message;
    errorModal.show();
}

function formatSize(bytes) {
    const units = ['B', 'KB', 'MB', 'GB'];
    let i = 0;
    while (bytes >= 1024 && i < units.length - 1) {
        bytes /= 1024;
        i++;
    }
    return bytes.toFixed(1) + ' ' + units[i];
}


async function withRetry(fn, {
    retries = 3,
    retryDelay = 3000,
    onRetry,
    onWait,
    signal
} = {}) {
    let lastError;
    let attempt = 1;
    const startTime = Date.now();
    const MAX_WAIT_TIME = 60000; // 60 seconds

    while (attempt <= retries) {
        if (signal && signal.aborted) throw new Error("Cancelled");

        try {
            return await fn();
        } catch (err) {
            lastError = err;

            if (err.message === "Cancelled" || (signal && signal.aborted)) throw err;

            // Handle Rate Limiting (429)
            if (err.status === 429) {
                const elapsed = Date.now() - startTime;
                if (elapsed < MAX_WAIT_TIME) {
                    if (onWait) onWait();
                    await new Promise(r => setTimeout(r, 5000));
                    continue; // "continue" doesn't increment 'attempt', so it retries indefinitely for 60s
                }
            }

            // Standard Retry Logic
            if (onRetry && attempt < retries) {
                onRetry(attempt, err);
            }

            if (err.status === 400 || err.status === 401) throw err;

            if (attempt < retries) {
                attempt++;
                await new Promise(r => setTimeout(r, retryDelay));
            } else {
                break;
            }
        }
    }
    throw lastError;
}

function getQueuedFileCount() {
    let count = 0;
    for (const entry of filesMap.values()) {
        if (!entry.removed) count++;
    }
    return count;
}

async function initUpload() {
    const btn = document.getElementById("uploadbutton");
    isUploadInProgress = true;
    btn.disabled = true;

    try {
        await startUpload();
    } catch (e) {
        console.error(e);
    } finally {
        isUploadInProgress = false;
        updateUploadButtonState();
    }
}

async function startUpload() {
    if (!IS_UNLIMITED_FILES && getQueuedFileCount() > maxFilesRemaining) {
        showModal("maxfiles");
        return;
    }

    for (const entry of filesMap.values()) {
        if (entry.removed || entry.status !== 'pending') {
            continue;
        }
        const {
            file,
            uuid,
            elements
        } = entry;

        entry.status = 'uploading';

        // Reset UI state for (re)attempt
        elements.progressBar.style.display = "";
        elements.progressText.style.color = "";
        let lastSpeedText = "";

        try {
            elements.progressText.textContent = t("pu_status_reserving");
            const serverUuid = await reserveChunk(elements);
            entry.serverUuid = serverUuid;

            elements.removeBtn.innerHTML = '<i class="bi bi-stop-circle text-danger"></i>';
            elements.removeBtn.title = t("pu_cancel_upload");

            let offset = 0;
            // do-while so that add chunk is run for 0byte files as well
            do {
                if (entry.controller.signal.aborted) return;
                const chunk = file.slice(offset, offset + CHUNK_SIZE);

                await withRetry(async () => {
                    return new Promise((resolve, reject) => {
                        const formData = new FormData();
                        formData.append("file", chunk);
                        formData.append("uuid", serverUuid);
                        formData.append("filesize", file.size);
                        formData.append("offset", offset);

                        const xhr = new XMLHttpRequest();
                        entry.xhr = xhr;
                        xhr.open("POST", UPLOAD_URL);
                        xhr.setRequestHeader("apikey", API_KEY);
                        xhr.setRequestHeader("fileRequestId", FILE_REQUEST_ID);

                        const startTime = Date.now();

                        // Listen for the cancel signal
                        const abortHandler = () => {
                            xhr.abort();
                            reject(new Error("Cancelled"));
                        };
                        entry.controller.signal.addEventListener('abort', abortHandler);

                        xhr.upload.onprogress = (event) => {
                            if (event.lengthComputable) {
                                const chunkOffset = offset + event.loaded;
                                const totalSize = file.size === 0 ? 1 : file.size;
                                const percent = Math.floor((chunkOffset / totalSize) * 100);

                                const duration = (Date.now() - startTime) / 1000;
                                if (duration > 0) {
                                    // Update the persistent lastSpeedText
                                    lastSpeedText = ` (${formatSize(event.loaded / duration)}/s)`;
                                }

                                elements.progressBar.value = chunkOffset;
                                elements.progressText.textContent = percent + "%" + lastSpeedText;
                            }
                        };

                        xhr.onload = async () => {
                            entry.controller.signal.removeEventListener('abort', abortHandler);
                            if (xhr.status >= 200 && xhr.status < 300) resolve();
                            else reject(await parseXhrError(xhr));
                        };

                        xhr.onerror = () => {
                            const err = new Error(t("status_server_error"));
                            err.status = xhr.status;
                            reject(err);
                        };

                        xhr.send(formData);
                    });
                }, {
                    signal: entry.controller.signal,
                    onWait: () => {
                        elements.progressText.textContent = t("pu_status_waiting_slot");
                    },
                    onRetry: (a, e) => {
                        elements.progressText.textContent = t("pu_status_retry", a, e.message) + lastSpeedText;
                    }
                });

                offset += chunk.size;
            } while (offset < file.size);

            await finaliseUpload(file, serverUuid, elements);

            entry.status = 'completed';
            elements.progressText.textContent = t("pu_status_completed");
            elements.item.style.opacity = "0.6";
            elements.removeBtn.remove(); // Remove button only on success

            filesMap.get(uuid).removed = true;
            maxFilesRemaining--;

            if (maxFilesRemaining === 0) showModal("alluploaded");

        } catch (err) {
            if (err.message === "Cancelled" || entry.controller.signal.aborted) {
                entry.status = 'pending';
                return;
            }

            entry.status = 'error';
            elements.progressText.textContent = err.message || t("pu_upload_failed");
            elements.progressText.style.color = "#ff6b6b";
            elements.progressBar.style.display = "none";

            elements.removeBtn.innerHTML = '<i class="bi bi-trash"></i>';
            elements.removeBtn.title = t("pu_remove_from_list");
        }
    }
}

async function parseXhrError(xhr) {
    // Wrap the XHR data into a format that parseErrorResponse expects
    const mockResponse = {
        ok: false,
        status: xhr.status,
        text: async () => xhr.responseText || `HTTP ${xhr.status}`
    };

    return await parseErrorResponse(mockResponse);
}

async function parseErrorResponse(response) {
    const text = await response.text();
    let data = null;
    try {
        data = JSON.parse(text);
    } catch {
        /* not JSON */
    }
    if (data && data.Result === "error") {
        let message;
        switch (data.ErrorCode) {
            case 9:
                message = t("pu_err_size_limit");
                break;
            case 14:
                message = t("pu_err_expired");
                showModal("expired");
                break;
            case 15:
                message = t("pu_err_max_files");
                showModal("maxfilesdynamic");
                break;
            case 16:
                message = t("pu_err_rate_limit");
                break;
            default:
                message = data.ErrorMessage || t("pu_err_unknown");
        }
        const err = new Error(message);
        err.status = response.status;
        err.code = data.ErrorCode;
        err.raw = data;
        return err;
    }
    // Fallback: plain text / non-JSON error
    const err = new Error(text || `HTTP ${response.status}`);
    err.status = response.status;
    return err;
}

async function reserveChunk(elements) {
    return withRetry(async () => {
        const response = await fetch(RESERVE_URL, {
            method: "POST",
            headers: {
                id: FILE_REQUEST_ID,
                apikey: API_KEY
            }
        });
        if (!response.ok) {
            throw await parseErrorResponse(response);
        }
        const data = await response.json();
        if (!data.Uuid) throw new Error(t("pu_err_invalid_reserve"));
        return data.Uuid;
    }, {
        onRetry: (a, e) => {
            elements.progressText.textContent = t("pu_status_retry", a, e.message);
        }
    });
}

async function finaliseUpload(file, uuid, elements) {
    await withRetry(async () => {
        const response = await fetch(COMPLETE_URL, {
            method: "POST",
            headers: {
                uuid,
                fileRequestId: FILE_REQUEST_ID,
                filename: encodeFilename(file.name),
                filesize: file.size,
                nonblocking: true,
                contenttype: file.type || "application/octet-stream",
                apikey: API_KEY
            }
        });
        if (!response.ok) {
            throw await parseErrorResponse(response);
        }
    }, {
        onRetry: (a, e) => {
            elements.progressText.textContent = t("pu_status_retry", a, e.message);
        }
    });
}

function encodeFilename(name) {
    return "base64:" + Base64.encode(name);
}



async function unreserve(uuid) {
    if (!uuid) return;
    try {
        await fetch(UNRESERVE_URL, {
            method: "POST",
            headers: {
                uuid: uuid,
                apikey: API_KEY,
                id: FILE_REQUEST_ID
            },
            keepalive: true // Crucial for calls during page unload
        });
    } catch (e) {
        console.error("Unreserve failed", e);
    }
}
