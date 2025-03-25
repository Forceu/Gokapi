// This file contains JS code for the API view
// All files named admin_*.js will be merged together and minimised by calling
// go generate ./...


function changeApiPermission(userId, permission, buttonId) {

    var indicator = document.getElementById(buttonId);
    if (indicator.classList.contains("perm-processing") || indicator.classList.contains("perm-nochange")) {
        return;
    }
    var wasGranted = indicator.classList.contains("perm-granted");
    indicator.classList.add("perm-processing");
    indicator.classList.remove("perm-granted");
    indicator.classList.remove("perm-notgranted");

    var modifier = "GRANT";
    if (wasGranted) {
        modifier = "REVOKE";
    }


    apiAuthModify(userId, permission, modifier)
        .then(data => {
            if (wasGranted) {
                indicator.classList.add("perm-notgranted");
            } else {
                indicator.classList.add("perm-granted");
            }
            indicator.classList.remove("perm-processing");
        })
        .catch(error => {
            if (wasGranted) {
                indicator.classList.add("perm-granted");
            } else {
                indicator.classList.add("perm-notgranted");
            }
            indicator.classList.remove("perm-processing");
            alert("Unable to set permission: " + error);
            console.error('Error:', error);
        });
}

function deleteApiKey(apiKey) {

    document.getElementById("delete-" + apiKey).disabled = true;

    apiAuthDelete(apiKey)
        .then(data => {
        document.getElementById("row-" + apiKey).classList.add("rowDeleting");
        setTimeout(() => {
            document.getElementById("row-" + apiKey).remove();
    }, 290);
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
            addRowApi(data.Id, data.PublicId);
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
        if (newName == "") {
        	newName = "Unnamed key";
        }
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




function addRowApi(apiKey, publicId) {

    let table = document.getElementById("apitable");
    let row = table.insertRow(0);
    
    row.id = "row-" + publicId;
    let cellCount = 0;
    let cellFriendlyName = row.insertCell(cellCount++);
    let cellId = row.insertCell(cellCount++);
    let cellLastUsed = row.insertCell(cellCount++);
    let cellPermissions = row.insertCell(cellCount++);
    let cellUserName;
    if (canViewOtherApiKeys) {
    	cellUserName= row.insertCell(cellCount++);
    	}
    let cellButtons= row.insertCell(cellCount++);
    
    if (canViewOtherApiKeys) {
    cellUserName.classList.add("newApiKey");
    cellUserName.innerText = userName;
    }

    cellFriendlyName.classList.add("newApiKey");
    cellId.classList.add("newApiKey");
    cellLastUsed.classList.add("newApiKey");
    cellPermissions.classList.add("newApiKey");
    cellPermissions.classList.add("prevent-select");
    cellButtons.classList.add("newApiKey");


    cellFriendlyName.innerText = "Unnamed key";
    cellFriendlyName.id = "friendlyname-" + publicId;
    cellFriendlyName.onclick = function() {
        addFriendlyNameChange(publicId);
    };
    cellId.innerHTML = '<div class="font-monospace">'+apiKey+'</div>';
    cellLastUsed.innerText = "Never";
    cellButtons.innerHTML = '<button type="button" data-clipboard-text="' + apiKey + '"  onclick="showToast(1000)" title="Copy API Key" class="copyurl btn btn-outline-light btn-sm"><i class="bi bi-copy"></i></button> <button id="delete-' + publicId + '" type="button" class="btn btn-outline-danger btn-sm" onclick="deleteApiKey(\'' + publicId + '\')" title="Delete"><i class="bi bi-trash3"></i></button>';
    cellPermissions.innerHTML = `
	    	<i id="perm_view_` + publicId + `" class="bi bi-eye perm-granted" title="List Uploads" onclick='changeApiPermission("` + publicId + `","PERM_VIEW", "perm_view_` + publicId + `");'></i>
	    	<i id="perm_upload_` + publicId + `" class="bi bi-file-earmark-arrow-up perm-granted" title="Upload" onclick='changeApiPermission("` + publicId + `","PERM_UPLOAD", "perm_upload_` + publicId + `");'></i>
	    	<i id="perm_edit_` + publicId + `" class="bi bi-pencil perm-granted" title="Edit Uploads" onclick='changeApiPermission("` + publicId + `","PERM_EDIT", "perm_edit_` + publicId + `");'></i>
	    	<i id="perm_delete_` + publicId + `" class="bi bi-trash3 perm-granted" title="Delete Uploads" onclick='changeApiPermission("` + publicId + `","PERM_DELETE", "perm_delete_` + publicId + `");'></i>
	    	<i id="perm_replace_` + publicId + `" class="bi bi-recycle perm-notgranted" title="Replace Uploads" onclick='changeApiPermission("` + publicId + `","PERM_REPLACE", "perm_replace_` + publicId + `");'></i>
	    	<i id="perm_users_` + publicId + `" class="bi bi-people perm-notgranted" title="Manage Users" onclick='changeApiPermission("` + publicId + `", "PERM_MANAGE_USERS", "perm_users_` + publicId + `");'></i>
	    	<i id="perm_logs_` + publicId + `" class="bi bi-card-list perm-notgranted" title="Manage System Logs" onclick='changeApiPermission("` + publicId + `", "PERM_MANAGE_LOGS", "perm_logs_` + publicId + `");'></i>
	    	<i id="perm_api_` + publicId + `" class="bi bi-sliders2 perm-notgranted" title="Manage API Keys" onclick='changeApiPermission("` + publicId + `","PERM_API_MOD", "perm_api_` + publicId + `");'></i>`;
   
    if (!canReplaceFiles) {
    	let cell = document.getElementById("perm_replace_"+publicId);
    	cell.classList.add("perm-unavailable");
    	cell.classList.add("perm-nochange");
    }
    if (!canManageUsers) {
    	let cell = document.getElementById("perm_users_"+publicId);
    	cell.classList.add("perm-unavailable");
    	cell.classList.add("perm-nochange");
    }

    setTimeout(() => {
        cellFriendlyName.classList.remove("newApiKey");
        cellId.classList.remove("newApiKey");
        cellLastUsed.classList.remove("newApiKey");
        cellPermissions.classList.remove("newApiKey");
        cellButtons.classList.remove("newApiKey");
    }, 700);

}
