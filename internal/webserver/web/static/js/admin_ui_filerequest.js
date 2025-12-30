// This file contains JS code for the API view
// All files named admin_*.js will be merged together and minimised by calling
// go generate ./...

function deleteFileRequest(requestId) {
    document.getElementById("delete-" + requestId).disabled = true;

    apiURequestDelete(requestId)
        .then(data => {
            document.getElementById("row-" + requestId).classList.add("rowDeleting");
            setTimeout(() => {
                document.getElementById("row-" + requestId).remove();
            }, 290);
        })
        .catch(error => {
            alert("Unable to delete file request: " + error);
            console.error('Error:', error);
        });
}

function deleteOrShowModal(requestId, requestName, count) {
    if (count === 0) {
        deleteFileRequest(requestId);
    } else {
        showDeleteFRequestModal(requestId, requestName, count);
    }
}

function showDeleteFRequestModal(requestId, requestName, count) {
    document.getElementById("deleteModalBodyName").innerText = requestName;
    document.getElementById("deleteModalBodyCount").innerText = count;
    $('#deleteModal').modal('show');

    document.getElementById("buttonDelete").onclick = function() {
        $('#deleteModal').modal('hide');
        deleteFileRequest(requestId);
    };
}


function newFileRequest() {
    loadFileRequestDefaults();
    document.getElementById("m_urequestlabel").innerText = "New File Request";
    $('#addEditModal').modal('show');

    document.getElementById("b_fr_save").onclick = function() {
        saveFileRequestDefaults();
        saveFileRequest();
        $('#addEditModal').modal('hide');
    };
}

function saveFileRequestDefaults() {
    if (document.getElementById("mc_maxfiles").checked) {
        localStorage.setItem("fr_maxfiles", document.getElementById("mi_maxfiles").value);
    } else {
        localStorage.setItem("fr_maxfiles", 0);
    }
    if (document.getElementById("mc_maxsize").checked) {
        localStorage.setItem("fr_maxsize", document.getElementById("mi_maxsize").value);
    } else {
        localStorage.setItem("fr_maxsize", 0);
    }
    if (document.getElementById("mc_expiry").checked) {
        let diff = document.getElementById("mi_expiry").value - Math.round(Date.now() / 1000);
        localStorage.setItem("fr_expiry", diff);
    } else {
        localStorage.setItem("fr_expiry", 0);
    }
}

function loadFileRequestDefaults() {
    const defaultMaxFiles = localStorage.getItem("fr_maxfiles");
    const defaultMaxSize = localStorage.getItem("fr_maxsize");
    let defaultExpiry = localStorage.getItem("fr_expiry");

    let defaultDate = new Date(Date.now() + Number((defaultExpiry) * 1000));
    defaultDate.setHours(12, 0, 0, 0);
    defaultExpiry = Math.floor(defaultDate.getTime() / 1000);

    setModalValues(0, "", defaultMaxFiles, defaultMaxSize, defaultExpiry);
}

function setModalValues(id, name, maxFiles, maxSize, expiry) {
    document.getElementById("freqId").value = id;

    if (name === null) {
        document.getElementById("mFriendlyName").value = "";
    } else {
        document.getElementById("mFriendlyName").value = name;
    }

    if (maxFiles === null || maxFiles == 0) {
        document.getElementById("mi_maxfiles").value = "1";
        document.getElementById("mi_maxfiles").disabled = true;
        document.getElementById("mc_maxfiles").checked = false;
    } else {
        document.getElementById("mi_maxfiles").value = maxFiles;
        document.getElementById("mi_maxfiles").disabled = false;
        document.getElementById("mc_maxfiles").checked = true;
    }

    if (maxSize === null || maxSize == 0) {
        document.getElementById("mi_maxsize").value = "10";
        document.getElementById("mi_maxsize").disabled = true;
        document.getElementById("mc_maxsize").checked = false;
    } else {
        document.getElementById("mi_maxsize").value = maxSize;
        document.getElementById("mi_maxsize").disabled = false;
        document.getElementById("mc_maxsize").checked = true;
    }

    if (expiry === null || expiry == 0) {
        const defaultDate = Math.floor(new Date(Date.now() + (14 * 24 * 60 * 60 * 1000)).getTime() / 1000);
        document.getElementById("mi_expiry").disabled = true;
        document.getElementById("mc_expiry").checked = false;
        document.getElementById("mi_expiry").value = defaultDate;
        createCalendar("mi_expiry", defaultDate);
    } else {
        document.getElementById("mi_expiry").value = expiry;
        document.getElementById("mi_expiry").disabled = false;
        document.getElementById("mc_expiry").checked = true;
        createCalendar("mi_expiry", expiry);
    }
}

function editFileRequest(id, name, maxFiles, maxSize, expiry) {
    setModalValues(id, name, maxFiles, maxSize, expiry);
    document.getElementById("m_urequestlabel").innerText = "Edit File Request";
    $('#addEditModal').modal('show');

    document.getElementById("b_fr_save").onclick = function() {
        saveFileRequest();
        $('#addEditModal').modal('hide');
    };
}


function saveFileRequest() {
    const buttonSave = document.getElementById("b_fr_save");
    const id = document.getElementById("freqId").value;
    const name = document.getElementById("mFriendlyName").value;
    let maxFiles = 0;
    let maxSize = 0;
    let expiry = 0;

    if (document.getElementById("mc_maxfiles").checked) {
        maxFiles = document.getElementById("mi_maxfiles").value;
    }
    if (document.getElementById("mc_maxsize").checked) {
        maxSize = document.getElementById("mi_maxsize").value;
    }
    if (document.getElementById("mc_expiry").checked) {
        expiry = document.getElementById("mi_expiry").value;
    }

    buttonSave.disabled = true;
    apiURequestSave(id, name, maxFiles, maxSize, expiry)
        .then(data => {
            document.getElementById("b_fr_save").disabled = false;
            insertOrReplaceFileRequest(data);
        })
        .catch(error => {
            alert("Unable to save file request: " + error);
            console.error('Error:', error);
            document.getElementById("b_fr_save").disabled = false;
        });
}

function insertOrReplaceFileRequest(jsonResult) {
    const tbody = document.getElementById("filerequesttable");
    let row = document.getElementById(`row-${jsonResult.id}`);

    if (row) {
        const user = document.getElementById(`cell-username-${jsonResult.id}`).innerText;
        row.replaceWith(createFileRequestRow(jsonResult, user));
    } else {
        let tr = createFileRequestRow(jsonResult, userName);
        tr.querySelectorAll('td').forEach((td) => {
            td.classList.add("newFileRequest");
            setTimeout(() => {
                td.classList.remove("newFileRequest");
            }, 700);
        });
        tbody.prepend(tr);
    }
}


function createFileRequestRow(jsonResult, user) {

    function tdText(text) {
        const td = document.createElement("td");
        td.textContent = text;
        return td;
    }

    function icon(classes) {
        const i = document.createElement("i");
        i.className = `bi ${classes}`;
        return i;
    }


    const tr = document.createElement("tr");
    tr.id = `row-${jsonResult.id}`;

    // Name
    tr.appendChild(tdText(jsonResult.name));
    // Uploaded files / Max files
    if (jsonResult.maxfiles == 0) {
        tr.appendChild(tdText(jsonResult.uploadedfiles));
    } else {
        tr.appendChild(tdText(`${jsonResult.uploadedfiles} / ${jsonResult.maxfiles}`));
    }
    // Total size
    tr.appendChild(tdText(getReadableSize(jsonResult.totalfilesize)));
    // Last upload
    tr.appendChild(tdText(formatTimestampWithNegative(jsonResult.lastupload, "None")));
    // Expiry
    tr.appendChild(tdText(formatFileRequestExpiry(jsonResult.expiry)));
    // Optional user column
    if (canViewOtherRequests) {
        let userTd = tdText(user);
        userTd.id = `cell-username-${jsonResult.id}`;
        tr.appendChild(userTd);
    }
    // Buttons
    const td = document.createElement("td");

    const group = document.createElement("div");
    group.className = "btn-group";
    group.role = "group";

    // Download
    const downloadBtn = document.createElement("button");
    downloadBtn.id = `download-${jsonResult.id}`;
    downloadBtn.type = "button";
    downloadBtn.className = "btn btn-outline-light btn-sm";
    downloadBtn.title = "Download all";

    if (jsonResult.uploadedfiles == 0) {
        downloadBtn.classList.add("disabled");
    }

    downloadBtn.appendChild(icon("bi-download"));

    // Edit
    const editBtn = document.createElement("button");
    editBtn.id = `edit-${jsonResult.id}`;
    editBtn.type = "button";
    editBtn.className = "btn btn-outline-light btn-sm";
    editBtn.title = "Edit request";
    editBtn.onclick = () =>
        editFileRequest(jsonResult.id, jsonResult.name, jsonResult.maxfiles, jsonResult.maxsize, jsonResult.expiry);

    editBtn.appendChild(icon("bi-pencil"));

    // Delete
    const deleteBtn = document.createElement("button");
    deleteBtn.id = `delete-${jsonResult.id}`;
    deleteBtn.type = "button";
    deleteBtn.className = "btn btn-outline-danger btn-sm";
    deleteBtn.title = "Delete";
    deleteBtn.onclick = () =>
        deleteOrShowModal(jsonResult.id, jsonResult.name, jsonResult.uploadedfiles);

    deleteBtn.appendChild(icon("bi-trash3"));

    group.append(downloadBtn, editBtn, deleteBtn);
    td.appendChild(group);
    tr.appendChild(td);
    return tr;
}
