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
    let currentName = cell.innerText;
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
        cell.innerText = newName;

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
    cell.innerText = "";
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
        cellUserName = row.insertCell(cellCount++);
    }
    let cellButtons = row.insertCell(cellCount++);

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
    cellId.innerText = apiKey;
    cellId.classList.add("font-monospace");
    cellLastUsed.innerText = "Never";


    // === Buttons Cell ===
    const copyButton = document.createElement('button');
    copyButton.type = 'button';
    copyButton.dataset.clipboardText = apiKey;
    copyButton.title = 'Copy API Key';
    copyButton.className = 'copyurl btn btn-outline-light btn-sm';
    copyButton.setAttribute('onclick', 'showToast(1000)');

    const copyIcon = document.createElement('i');
    copyIcon.className = 'bi bi-copy';
    copyButton.appendChild(copyIcon);

    const deleteButton = document.createElement('button');
    deleteButton.type = 'button';
    deleteButton.id = `delete-${publicId}`;
    deleteButton.title = 'Delete';
    deleteButton.className = 'btn btn-outline-danger btn-sm';
    deleteButton.setAttribute('onclick', `deleteApiKey('${publicId}')`);

    const deleteIcon = document.createElement('i');
    deleteIcon.className = 'bi bi-trash3';
    deleteButton.appendChild(deleteIcon);

    cellButtons.appendChild(copyButton);
    cellButtons.appendChild(document.createTextNode(' ')); // space between buttons
    cellButtons.appendChild(deleteButton);

    // === Permissions Cell ===
    const perms = [{
            perm: 'PERM_VIEW',
            icon: 'bi-eye',
            granted: true,
            title: 'List Uploads'
        },
        {
            perm: 'PERM_UPLOAD',
            icon: 'bi-file-earmark-arrow-up',
            granted: true,
            title: 'Upload'
        },
        {
            perm: 'PERM_EDIT',
            icon: 'bi-pencil',
            granted: true,
            title: 'Edit Uploads'
        },
        {
            perm: 'PERM_DELETE',
            icon: 'bi-trash3',
            granted: true,
            title: 'Delete Uploads'
        },
        {
            perm: 'PERM_REPLACE',
            icon: 'bi-recycle',
            granted: false,
            title: 'Replace Uploads'
        },
        {
            perm: 'PERM_MANAGE_USERS',
            icon: 'bi-people',
            granted: false,
            title: 'Manage Users'
        },
        {
            perm: 'PERM_MANAGE_LOGS',
            icon: 'bi-card-list',
            granted: false,
            title: 'Manage System Logs'
        },
        {
            perm: 'PERM_API_MOD',
            icon: 'bi-sliders2',
            granted: false,
            title: 'Manage API Keys'
        }
    ];

    perms.forEach(({
        perm,
        icon,
        granted,
        title
    }) => {
        const i = document.createElement('i');
        const id = `perm_${perm.toLowerCase().replace('perm_', '')}_${publicId}`;
        i.id = id;
        i.className = `bi ${icon} ${granted ? 'perm-granted' : 'perm-notgranted'}`;
        i.title = title;
        i.setAttribute('onclick', `changeApiPermission("${publicId}","${perm}", "${id}");`);
        cellPermissions.appendChild(i);
        cellPermissions.appendChild(document.createTextNode(' '));
    });


    if (!canReplaceFiles) {
        let cell = document.getElementById("perm_replace_" + publicId);
        cell.classList.add("perm-unavailable");
        cell.classList.add("perm-nochange");
    }
    if (!canManageUsers) {
        let cell = document.getElementById("perm_users_" + publicId);
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
