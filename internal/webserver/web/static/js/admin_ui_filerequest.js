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
