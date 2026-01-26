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



function showDeleteUserModal(userId, userEmail) {
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


function showResetPwModal(userid, name) {
    // Cloning removes any previous values or form validation
    let originalModal = $('#resetPasswordModal').clone();
    $("#resetPasswordModal").on('hide.bs.modal', function() {
        $('#resetPasswordModal').remove();
        let myClone = originalModal.clone();
        $('body').append(myClone);
    });

    document.getElementById("l_userpwreset").innerText = name;
    let button = document.getElementById("resetPasswordButton");
    button.onclick = function() {
        resetPw(userid, document.getElementById("generateRandomPassword").checked);
    };
    $('#resetPasswordModal').modal('show');
}

function resetPw(userid, newPw) {
    let button = document.getElementById("resetPasswordButton");
    document.getElementById("resetPasswordButton").disabled = true;
    apiUserResetPassword(userid, newPw)
        .then(data => {
            if (!newPw) {
                $('#resetPasswordModal').modal('hide');
                showToast(1000, 'Password change requirement set successfully')
                return;
            }
            button.style.display = 'none';
            document.getElementById("cancelPasswordButton").style.display = 'none';
            document.getElementById("formentryReset").style.display = 'none';
            document.getElementById("randomPasswordContainer").style.display = 'block';
            document.getElementById("closeModalResetPw").style.display = 'block';
            document.getElementById("l_returnedPw").innerText = data.password;
            document.getElementById("copypwclip").onclick = function() {
                // For some reason ClipboardJs is not working on the user PW reset modal, even when initilising again. Manually writing to clipboard
                navigator.clipboard.writeText(data.password);
                showToast(1000, "Password copied to clipboard");
            }
        })
        .catch(error => {
            alert("Unable to reset user password: " + error);
            console.error('Error:', error);
            button.disabled = false;
        });
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
                $('#newUserModal').modal('hide');
                addRowUser(data.id, data.name, data.permissions);
                console.log(data);
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



const PermissionDefinitions = [
    {
        key: "UserPermGuestUploads",
        bit: 1 << 8,
        icon: "bi bi-box-arrow-in-down",
        title: "Create file requests",
        htmlId: userid => `perm_guest_upload_${userid}`,
        apiName: "PERM_GUEST_UPLOAD"
    },
    {
        key: "UserPermReplaceUploads",
        bit: 1 << 0,
        icon: "bi bi-recycle",
        title: "Replace own uploads",
        htmlId: userid => `perm_replace_${userid}`,
        apiName: "PERM_REPLACE"
    },
    {
        key: "UserPermListOtherUploads",
        bit: 1 << 1,
        icon: "bi bi-eye",
        title: "List other uploads",
        htmlId: userid => `perm_list_${userid}`,
        apiName: "PERM_LIST"
    },
    {
        key: "UserPermEditOtherUploads",
        bit: 1 << 2,
        icon: "bi bi-pencil",
        title: "Edit other uploads",
        htmlId: userid => `perm_edit_${userid}`,
        apiName: "PERM_EDIT"
    },
    {
        key: "UserPermDeleteOtherUploads",
        bit: 1 << 4,
        icon: "bi bi-trash3",
        title: "Delete other uploads",
        htmlId: userid => `perm_delete_${userid}`,
        apiName: "PERM_DELETE"
    },
    {
        key: "UserPermReplaceOtherUploads",
        bit: 1 << 3,
        icon: "bi bi-arrow-left-right",
        title: "Replace other uploads",
        htmlId: userid => `perm_replace_other_${userid}`,
        apiName: "PERM_REPLACE_OTHER"
    },
    {
        key: "UserPermManageLogs",
        bit: 1 << 5,
        icon: "bi bi-card-list",
        title: "Manage system logs",
        htmlId: userid => `perm_logs_${userid}`,
        apiName: "PERM_LOGS"
    },
    {
        key: "UserPermManageUsers",
        bit: 1 << 7,
        icon: "bi bi-people",
        title: "Manage users",
        htmlId: userid => `perm_users_${userid}`,
        apiName: "PERM_USERS"
    },
    {
        key: "UserPermManageApiKeys",
        bit: 1 << 6,
        icon: "bi bi-sliders2",
        title: "Manage API keys",
        htmlId: userid => `perm_api_${userid}`,
        apiName: "PERM_API"
    }
];

function hasPermission(userPermissions, permissionBit) {
    return (userPermissions & permissionBit) !== 0;
}


function addRowUser(userid, name, permissions) {

    userid = sanitizeUserId(userid);

    let table = document.getElementById("usertable");
    let row = table.insertRow(1);
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

    // Create one button group
    const btnGroup = document.createElement("div");
    btnGroup.className = "btn-group";
    btnGroup.setAttribute("role", "group");

    // Password reset button (optional)
    if (isInternalAuth) {
        const btnResetPw = document.createElement("button");
        btnResetPw.id = `pwchange-${userid}`;
        btnResetPw.type = "button";
        btnResetPw.className = "btn btn-outline-light btn-sm";
        btnResetPw.title = "Reset Password";
        btnResetPw.onclick = () => showResetPwModal(userid, name);
        btnResetPw.innerHTML = `<i class="bi bi-key-fill"></i>`;
        btnGroup.appendChild(btnResetPw);
    }

    // Promote button
    const btnPromote = document.createElement("button");
    btnPromote.id = `changeRank_${userid}`;
    btnPromote.type = "button";
    btnPromote.className = "btn btn-outline-light btn-sm";
    btnPromote.title = "Promote User";
    btnPromote.onclick = () => changeRank(userid, 'ADMIN', `changeRank_${userid}`);
    btnPromote.innerHTML = `<i class="bi bi-chevron-double-up"></i>`;
    btnGroup.appendChild(btnPromote);

    // Delete button
    const btnDelete = document.createElement("button");
    btnDelete.id = `delete-${userid}`;
    btnDelete.type = "button";
    btnDelete.className = "btn btn-outline-danger btn-sm";
    btnDelete.title = "Delete";
    btnDelete.onclick = () => showDeleteUserModal(userid, name);
    btnDelete.innerHTML = `<i class="bi bi-trash3"></i>`;
    btnGroup.appendChild(btnDelete);

    // Insert button group into cellActions
    cellActions.innerHTML = '';
    cellActions.appendChild(btnGroup);

    // Permissions
     cellPermissions.innerHTML = PermissionDefinitions.map(perm => {
        const granted = hasPermission(permissions, perm.bit)
            ? "perm-granted"
            : "perm-notgranted";

        const id = perm.htmlId(userid);

        return `
        <i id="${id}"
           class="${perm.icon} ${granted}"
           title="${perm.title}"
           onclick='changeUserPermission(${userid}, "${perm.apiName}", "${id}")'>
        </i>`;
    }).join("");

    setTimeout(() => {

        cellName.classList.remove("newUser");
        cellGroup.classList.remove("newUser");
        cellLastOnline.classList.remove("newUser");
        cellUploads.classList.remove("newUser");
        cellPermissions.classList.remove("newUser");
        cellActions.classList.remove("newUser");
    }, 700);
}

function sanitizeUserId(id) {
    const numericId = id.toString().trim();
    if (!/^\d+$/.test(numericId)) {
        throw new Error("Invalid ID: must contain only digits.");
    }
    return numericId;
}
