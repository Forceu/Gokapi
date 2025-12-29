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
    resetURModal();
    loadFileRequestDefaults();
    document.getElementById("m_urequestlabel").innerText = "New File Request";
    $('#addEditModal').modal('show');

    document.getElementById("b_fr_save").onclick = function() {
        saveFileRequestDefaults();
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
    const defaultExpiry = localStorage.getItem("fr_expiry");

    if (defaultMaxFiles !== null) {
        if (defaultMaxFiles == 0) {
            document.getElementById("mi_maxfiles").value = "1";
            document.getElementById("mi_maxfiles").disabled = true;
            document.getElementById("mc_maxfiles").checked = false;
        } else {
            document.getElementById("mi_maxfiles").value = defaultMaxFiles;
            document.getElementById("mi_maxfiles").disabled = false;
            document.getElementById("mc_maxfiles").checked = true;
        }
    }
    if (defaultMaxSize !== null) {
        if (defaultMaxSize == 0) {
            document.getElementById("mi_maxsize").value = "10";
            document.getElementById("mi_maxsize").disabled = true;
            document.getElementById("mc_maxsize").checked = false;
        } else {
            document.getElementById("mi_maxsize").value = defaultMaxSize;
            document.getElementById("mi_maxsize").disabled = false;
            document.getElementById("mc_maxsize").checked = true;
        }
    }
    if (defaultExpiry !== null) {
        if (defaultExpiry == 0) {
            document.getElementById("mi_expiry").value = "1";
            document.getElementById("mi_expiry").disabled = true;
            document.getElementById("mc_expiry").checked = false;
        } else {
            let defaultDate = new Date(Date.now() + (defaultExpiry * 1000));
            defaultDate.setHours(12, 0, 0, 0);
            document.getElementById("mi_expiry").value = defaultDate;
            document.getElementById("mi_expiry").disabled = false;
            document.getElementById("mc_expiry").checked = true;
            createCalendar("mi_expiry", Math.floor(defaultDate.getTime() / 1000));
        }
    }
}

function editFileRequest() {
    resetURModal();
    document.getElementById("m_urequestlabel").innerText = "New File Request";
    $('#addEditModal').modal('show');

    document.getElementById("b_fr_save").onclick = function() {
        $('#addEditModal').modal('hide');
    };
}

function resetURModal() {
    const defaultDate = Math.floor(new Date(Date.now() + (14 * 24 * 60 * 60 * 1000)).getTime() / 1000);
    document.getElementById("mFriendlyName").value = "";
    document.getElementById("mi_maxfiles").value = "1";
    document.getElementById("mi_maxfiles").disabled = true;
    document.getElementById("mi_maxsize").value = "10";
    document.getElementById("mi_expiry").value = defaultDate;
    document.getElementById("mi_maxfiles").disabled = true;
    document.getElementById("mi_maxsize").disabled = true;
    document.getElementById("mi_expiry").disabled = true;
    document.getElementById("mc_maxfiles").checked = false;
    document.getElementById("mc_maxsize").checked = false;
    document.getElementById("mc_expiry").checked = false;
    createCalendar("mi_expiry", defaultDate);
}
