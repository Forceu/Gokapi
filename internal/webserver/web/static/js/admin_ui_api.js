// This file contains JS code for the API view
// All files named admin_*.js will be merged together and minimised by calling
// go generate ./...


function changeApiPermission(apiKey, permission, buttonId) {

    var indicator = document.getElementById(buttonId);
    if (indicator.classList.contains("apiperm-processing")) {
        return;
    }
    var wasGranted = indicator.classList.contains("apiperm-granted");
    indicator.classList.add("apiperm-processing");
    indicator.classList.remove("apiperm-granted");
    indicator.classList.remove("apiperm-notgranted");

    var modifier = "GRANT";
    if (wasGranted) {
        modifier = "REVOKE";
    }


    apiAuthModify(apiKey, permission, modifier)
        .then(data => {
            if (wasGranted) {
                indicator.classList.add("apiperm-notgranted");
            } else {
                indicator.classList.add("apiperm-granted");
            }
            indicator.classList.remove("apiperm-processing");
        })
        .catch(error => {
            if (wasGranted) {
                indicator.classList.add("apiperm-granted");
            } else {
                indicator.classList.add("apiperm-notgranted");
            }
            indicator.classList.remove("apiperm-processing");
            alert("Unable to set permission: " + error);
            console.error('Error:', error);
        });
}

function deleteApiKey(apiKey) {

    document.getElementById("delete-" + apiKey).disabled = true;

    apiAuthDelete(apiKey)
        .then(data => {
            document.getElementById("row-" + apiKey).remove();
        })
        .catch(error => {
            alert("Unable to delete API key: " + error);
            console.error('Error:', error);
        });
}



function newApiKey() {
    document.getElementById("button-newapi").disabled = true;
    apiAuthCreate()
        .then(data => {
            addRowApi(data.Id);
            document.getElementById("button-newapi").disabled = false;
        })
        .catch(error => {
            alert("Unable to create API key: " + error);
            console.error('Error:', error);
        });
}




function addFriendlyNameChange(apiKey) {
    let cell = document.getElementById("friendlyname-" + apiKey);
    if (cell.classList.contains("isBeingEdited"))
        return;
    cell.classList.add("isBeingEdited");
    let currentName = cell.innerHTML;
    let input = document.createElement("input");
    input.size = 5;
    input.value = currentName;
    let allowEdit = true;

    let submitEntry = function() {
        if (!allowEdit)
            return;
        allowEdit = false;
        let newName = input.value;
        cell.innerHTML = newName;

        cell.classList.remove("isBeingEdited");

        apiAuthFriendlyName(apiKey, newName)
            .catch(error => {
                alert("Unable to save name: " + error);
                console.error('Error:', error);
            });
    };

    input.onblur = submitEntry;
    input.addEventListener("keyup", function(event) {
        // Enter
        if (event.keyCode === 13) {
            event.preventDefault();
            submitEntry();
        }
    });
    cell.innerHTML = "";
    cell.appendChild(input);
    input.focus();
}




function addRowApi(apiKey) {

    let table = document.getElementById("apitable");
    let row = table.insertRow(0);
    row.id = "row-" + apiKey;
    let cellFriendlyName = row.insertCell(0);
    let cellId = row.insertCell(1);
    let cellLastUsed = row.insertCell(2);
    let cellPermissions = row.insertCell(3);
    let cellButtons = row.insertCell(4);
    let cellEmpty = row.insertCell(5);

    cellFriendlyName.classList.add("newApiKey");
    cellId.classList.add("newApiKey");
    cellLastUsed.classList.add("newApiKey");
    cellPermissions.classList.add("newApiKey");
    cellButtons.classList.add("newApiKey");
    cellEmpty.classList.add("newApiKey");


    cellFriendlyName.innerText = "Unnamed key";
    cellFriendlyName.id = "friendlyname-" + apiKey;
    cellFriendlyName.onclick = function() {
        addFriendlyNameChange(apiKey);
    };
    cellId.innerText = apiKey;
    cellLastUsed.innerText = "Never";
    cellButtons.innerHTML = '<button type="button" data-clipboard-text="' + apiKey + '"  onclick="showToast()" title="Copy API Key" class="copyurl btn btn-outline-light btn-sm"><i class="bi bi-copy"></i></button> <button id="delete-' + apiKey + '" type="button" class="btn btn-outline-danger btn-sm" onclick="deleteApiKey(\'' + apiKey + '\')" title="Delete"><i class="bi bi-trash3"></i></button>';
    cellPermissions.innerHTML = `
	    	<i id="perm_view_` + apiKey + `" class="bi bi-eye apiperm-granted" title="List Uploads" onclick='changeApiPermission("` + apiKey + `","PERM_VIEW", "perm_view_` + apiKey + `");'></i>
	    	<i id="perm_upload_` + apiKey + `" class="bi bi-file-earmark-arrow-up apiperm-granted" title="Upload" onclick='changeApiPermission("` + apiKey + `","PERM_UPLOAD", "perm_upload_` + apiKey + `");'></i>
	    	<i id="perm_edit_` + apiKey + `" class="bi bi-pencil apiperm-granted" title="Edit Uploads" onclick='changeApiPermission("` + apiKey + `","PERM_EDIT", "perm_edit_` + apiKey + `");'></i>
	    	<i id="perm_delete_` + apiKey + `" class="bi bi-trash3 apiperm-granted" title="Delete Uploads" onclick='changeApiPermission("` + apiKey + `","PERM_DELETE", "perm_delete_` + apiKey + `");'></i>
	    	<i id="perm_replace_` + apiKey + `" class="bi bi-recycle apiperm-notgranted" title="Replace Uploads" onclick='changeApiPermission("` + apiKey + `","PERM_REPLACE", "perm_replace_` + apiKey + `");'></i>
	    	<i id="perm_api_` + apiKey + `" class="bi bi-sliders2 apiperm-notgranted" title="Manage API Keys" onclick='changeApiPermission("` + apiKey + `","PERM_API_MOD", "perm_api_` + apiKey + `");'></i>`;

    setTimeout(() => {
        cellFriendlyName.classList.remove("newApiKey");
        cellId.classList.remove("newApiKey");
        cellLastUsed.classList.remove("newApiKey");
        cellPermissions.classList.remove("newApiKey");
        cellButtons.classList.remove("newApiKey");
        cellEmpty.classList.remove("newApiKey");
    }, 700);

}
