// This file contains JS code for the paste view.
// All files named admin_*.js will be merged together and minimised by calling
// go generate ./...

function pasteToggleDownloads(checkbox) {
    document.getElementById("paste-downloads").disabled = !checkbox.checked;
}

function pasteToggleExpiry(checkbox) {
    document.getElementById("paste-expiry").disabled = !checkbox.checked;
}

function pasteTogglePassword(checkbox) {
    document.getElementById("paste-password").disabled = !checkbox.checked;
}

function submitPaste() {
    const content = document.getElementById("paste-content").value.trim();
    if (!content) {
        alert("Please enter some content before creating a paste.");
        return;
    }

    const title        = document.getElementById("paste-title").value.trim();
    const limitViews   = document.getElementById("paste-enable-downloads").checked;
    const limitExpiry  = document.getElementById("paste-enable-expiry").checked;
    const usePassword  = document.getElementById("paste-enable-password").checked;

    const allowedDownloads = limitViews  ? parseInt(document.getElementById("paste-downloads").value, 10) : 0;
    const expiryDays       = limitExpiry ? parseInt(document.getElementById("paste-expiry").value,    10) : 0;
    const password         = usePassword ? document.getElementById("paste-password").value : "";

    apiAddPaste(content, title, allowedDownloads, expiryDays, password)
        .then(data => {
            const info = data.FileInfo;
            pasteInsertRow(info);
            document.getElementById("paste-content").value = "";
            document.getElementById("paste-title").value   = "";
            pasteCopyUrl(info.UrlDownload, info.Id);
        })
        .catch(error => {
            alert("Failed to create paste: " + error);
            console.error("Error:", error);
        });
}

function pasteInsertRow(info) {
    const tbody = document.getElementById("paste-tbody");
    const row   = document.createElement("tr");
    row.id      = "pasterow-" + info.Id;

    const views = info.UnlimitedDownloads ? "Unlimited" : info.DownloadsRemaining;

    row.innerHTML = `
        <td>${escapeHtml(info.Name)}</td>
        <td><span id="paste-created-${info.Id}"></span></td>
        <td><span id="paste-expiry-${info.Id}"></span></td>
        <td>${escapeHtml(String(views))}</td>
        <td>
          <div class="btn-group" role="group">
            <button type="button" class="btn btn-outline-light btn-sm" title="Copy URL"
              onclick="pasteCopyUrl('${escapeHtml(info.UrlDownload)}', '${escapeHtml(info.Id)}')">
              <i class="bi bi-copy"></i>
            </button>
            <button type="button" class="btn btn-outline-danger btn-sm" title="Delete"
              onclick="pasteDelete('${escapeHtml(info.Id)}')">
              <i class="bi bi-trash3"></i>
            </button>
          </div>
        </td>`;

    tbody.prepend(row);
    insertDateWithNegative(info.UploadDate, "paste-created-" + info.Id, "Unknown");

    if (info.UnlimitedTime) {
        document.getElementById("paste-expiry-" + info.Id).innerText = "Never";
    } else {
        insertFileRequestExpiry(info.ExpireAt, "paste-expiry-" + info.Id);
    }
}

function pasteCopyUrl(url, id) {
    navigator.clipboard.writeText(url).then(() => {
        const toastEl = document.getElementById("paste-toast");
        document.getElementById("paste-toast-body").innerText = "URL copied to clipboard!";
        bootstrap.Toast.getOrCreateInstance(toastEl).show();
    }).catch(() => {
        prompt("Copy this URL:", url);
    });
}

function pasteDelete(id) {
    if (!confirm("Delete this paste?")) {
        return;
    }
    apiFilesDelete(id, 0)
        .then(() => {
            const row = document.getElementById("pasterow-" + id);
            if (row) {
                row.classList.add("rowDeleting");
                setTimeout(() => row.remove(), 290);
            }
        })
        .catch(error => {
            alert("Failed to delete paste: " + error);
            console.error("Error:", error);
        });
}

function escapeHtml(str) {
    return String(str)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
}
