// This file contains JS code for the User view
// All files named admin_*.js will be merged together and minimised by calling
// go generate ./...


function changeUserPermission(userId, permission, buttonId) {

    let indicator = document.getElementById(buttonId);
    if (indicator.classList.contains("perm-processing") || indicator.classList.contains("perm-nochange")) {
        return;
    }
    let wasGranted = indicator.classList.contains("perm-granted");
    indicator.classList.add("perm-processing");
    indicator.classList.remove("perm-granted");
    indicator.classList.remove("perm-notgranted");

    let modifier = "GRANT";
    if (wasGranted) {
        modifier = "REVOKE";
    }

    if (permission == "PERM_REPLACE_OTHER" && !wasGranted) {
        hasNotPermissionReplace = document.getElementById("perm_replace_" + userId).classList.contains("perm-notgranted");
        if (hasNotPermissionReplace) {
            showToast(2000, "Also granting permission to replace own files");
            changeUserPermission(userId, "PERM_REPLACE", "perm_replace_" + userId);
        }
    }
    if (permission == "PERM_REPLACE" && wasGranted) {
        hasPermissionReplaceOthers = document.getElementById("perm_replace_other_" + userId).classList.contains("perm-granted");
        if (hasPermissionReplaceOthers) {
            showToast(2000, "Also revoking permission to replace files of other users");
            changeUserPermission(userId, "PERM_REPLACE_OTHER", "perm_replace_other_" + userId);
        }
    }


    apiUserModify(userId, permission, modifier)
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



function changeRank(userId, newRank, buttonId) {

    let indicator = document.getElementById(buttonId);
    if (indicator.disabled) {
        return;
    }
    indicator.disabled = true;

    apiUserChangeRank(userId, newRank)
        .then(data => {
            location.reload();
        })
        .catch(error => {
            indicator.disabled = false;
            alert("Unable to change rank: " + error);
            console.error('Error:', error);
        });
}



function showDeleteModal(userId, userEmail) {
    let checkboxDelete = document.getElementById("checkboxDelete");
    checkboxDelete.checked = false;
    document.getElementById("deleteModalBody").innerText = userEmail;
    $('#deleteModal').modal('show');

    document.getElementById("buttonDelete").onclick = function() {
        apiUserDelete(userId, checkboxDelete.checked)
            .then(data => {
                $('#deleteModal').modal('hide');
                document.getElementById("row-" + userId).classList.add("rowDeleting");
		setTimeout(() => {
		    document.getElementById("row-" + userId).remove();
	    }, 290);
            })
            .catch(error => {
                alert("Unable to delete user: " + error);
                console.error('Error:', error);
            });
    };
}


function showAddUserModal() {
    // Cloning removes any previous values or form validation
    let originalModal = $('#newUserModal').clone();
    $("#newUserModal").on('hide.bs.modal', function() {
        $('#newUserModal').remove();
        let myClone = originalModal.clone();
        $('body').append(myClone);
    });
    $('#newUserModal').modal('show');
}


function showResetPwModal(userid) {
//TODO
    $('#resetPasswordModal').modal('show');
}


function addNewUser() {
    let button = document.getElementById("mb_addUser");
    button.disabled = true;
    let form = document.getElementById('newUserForm');
    if (!form.checkValidity()) {
        form.classList.add('was-validated');
        button.disabled = false;
    } else {
        let editName = document.getElementById("e_userName");
        apiUserCreate(editName.value.trim())
            .then(data => {
            console.log(data);
                $('#newUserModal').modal('hide');
                addRowUser(data.id, data.name);
            })
            .catch(error => {
                if (error.message == "duplicate") {
                    alert("A user already exists with that name");
                    button.disabled = false;
                } else {
                    alert("Unable to create user: " + error);
                    console.error('Error:', error);
                    button.disabled = false;
                }
            });
    }
}




function addRowUser(userid, name) {

    let table = document.getElementById("usertable");
    let row = table.insertRow(0);
    row.id = "row-" + userid;
    let cellName = row.insertCell(0);
    let cellGroup = row.insertCell(1);
    let cellLastOnline = row.insertCell(2);
    let cellUploads = row.insertCell(3);
    let cellPermissions = row.insertCell(4);
    let cellActions = row.insertCell(5);

    cellName.classList.add("newUser");
    cellGroup.classList.add("newUser");
    cellLastOnline.classList.add("newUser");
    cellUploads.classList.add("newUser");
    cellPermissions.classList.add("newUser");
    cellActions.classList.add("newUser");


    cellName.innerText = name;
    cellGroup.innerText = "User";
    cellLastOnline.innerText = "Never";
    cellUploads.innerText = "0";
    cellActions.innerHTML = '<button id="changeRank_'+userid+'" type="button" onclick="changeRank( '+userid+' , \'ADMIN\', \'changeRank_'+userid+'\')" title="Promote User" class="btn btn-outline-light btn-sm"><i class="bi bi-chevron-double-up"></i></button>&nbsp;<button id="delete-'+userid+'" type="button" class="btn btn-outline-danger btn-sm"  onclick="showDeleteModal('+userid+', \''+name+'\')" title="Delete"><i class="bi bi-trash3"></i></button>';
    
    cellPermissions.innerHTML = `
    <i id="perm_replace_`+userid+`" class="bi bi-recycle perm-notgranted " title="Replace own uploads" onclick='changeUserPermission(`+userid+`,"PERM_REPLACE", "perm_replace_`+userid+`");'></i>
		
		<i id="perm_list_`+userid+`" class="bi bi-eye perm-notgranted " title="List other uploads" onclick='changeUserPermission(`+userid+`,"PERM_LIST", "perm_list_`+userid+`");'></i>
		
		<i id="perm_edit_`+userid+`" class="bi bi-pencil perm-notgranted " title="Edit other uploads" onclick='changeUserPermission(`+userid+`,"PERM_EDIT", "perm_edit_`+userid+`");'></i>
		
		<i id="perm_delete_`+userid+`" class="bi bi-trash3 perm-notgranted " title="Delete other uploads" onclick='changeUserPermission(`+userid+`,"PERM_DELETE", "perm_delete_`+userid+`");'></i>
		
		<i id="perm_replace_other_`+userid+`" class="bi bi-arrow-left-right perm-notgranted " title="Replace other uploads" onclick='changeUserPermission(`+userid+`,"PERM_REPLACE_OTHER", "perm_replace_other_`+userid+`");'></i>
		
		<i id="perm_logs_`+userid+`" class="bi bi-card-list perm-notgranted " title="Manage system logs" onclick='changeUserPermission(`+userid+`,"PERM_LOGS", "perm_logs_`+userid+`");'></i>

		<i id="perm_users_`+userid+`" class="bi bi-people perm-notgranted " title="Manage users" onclick='changeUserPermission(`+userid+`,"PERM_USERS", "perm_users_`+userid+`");'></i>

		<i id="perm_api_`+userid+`" class="bi bi-sliders2 perm-notgranted " title="Manage API keys" onclick='changeUserPermission(`+userid+`,"PERM_API", "perm_api_`+userid+`");'></i>`;

    setTimeout(() => {
    
    cellName.classList.remove("newUser");
    cellGroup.classList.remove("newUser");
    cellLastOnline.classList.remove("newUser");
    cellUploads.classList.remove("newUser");
    cellPermissions.classList.remove("newUser");
    cellActions.classList.remove("newUser");
    }, 700);

}
