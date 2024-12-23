// This file contains JS code to connect to the API
// All files named admin_*.js will be merged together and minimised by calling
// go generate ./...

// /auth

async function apiAuthModify(apiKey, permission, modifier) {
    const apiUrl = './api/auth/modify';

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': systemKey,
            'apiKeyToModify': apiKey,
            'permission': permission,
            'permissionModifier': modifier

        },
    };

    try {
        const response = await fetch(apiUrl, requestOptions);
        if (!response.ok) {
            throw new Error(`Request failed with status: ${response.status}`);
        }
    } catch (error) {
        console.error("Error in apiAuthModify:", error);
        throw error;
    }
}


async function apiAuthFriendlyName(apiKey, newName) {
    const apiUrl = './api/auth/friendlyname';

    const requestOptions = {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json',
            'apikey': systemKey,
            'apiKeyToModify': apiKey,
            'friendlyName': newName

        },
    };

    try {
        const response = await fetch(apiUrl, requestOptions);
        if (!response.ok) {
            throw new Error(`Request failed with status: ${response.status}`);
        }
    } catch (error) {
        console.error("Error in apiAuthModify:", error);
        throw error;
    }
}


async function apiAuthDelete(apiKey) {
    const apiUrl = './api/auth/delete';

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': systemKey,
            'apiKeyToModify': apiKey,
        },
    };

    try {
        const response = await fetch(apiUrl, requestOptions);
        if (!response.ok) {
            throw new Error(`Request failed with status: ${response.status}`);
        }
    } catch (error) {
        console.error("Error in apiAuthDelete:", error);
        throw error;
    }
}


async function apiAuthCreate() {
    const apiUrl = './api/auth/create';

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': systemKey,
            'basicPermissions': 'true'
        },
    };


    try {
        const response = await fetch(apiUrl, requestOptions);
        if (!response.ok) {
            throw new Error(`Request failed with status: ${response.status}`);
        }
        const data = await response.json();
        return data;
    } catch (error) {
        console.error("Error in apiAuthCreate:", error);
        throw error;
    }
}



// /files


async function apiFilesReplace(id, newId) {
    const apiUrl = './api/files/replace';

    const requestOptions = {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json',
            'id': id,
            'apikey': systemKey,
            'idNewContent': newId,
            'deleteNewFile': false
        },
    };

    try {
        const response = await fetch(apiUrl, requestOptions);
        if (!response.ok) {
            throw new Error(`Request failed with status: ${response.status}`);
        }
        const data = await response.json();
        return data;
    } catch (error) {
        console.error("Error in apiFilesReplace:", error);
        throw error;
    }
}

async function apiFilesListById(fileId) {
    const apiUrl = './api/files/list/' + fileId;
    const requestOptions = {
        method: 'GET',
        headers: {
            'Content-Type': 'application/json',
            'apikey': systemKey,

        },
    };

    try {
        const response = await fetch(apiUrl, requestOptions);
        if (!response.ok) {
            throw new Error(`Request failed with status: ${response.status}`);
        }
        const data = await response.json();
        return data;
    } catch (error) {
        console.error("Error in apiFilesListById:", error);
        throw error;
    }
}


async function apiFilesModify(id, allowedDownloads, expiry, password, originalPw) {
    const apiUrl = './api/files/modify';

    const requestOptions = {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json',
            'id': id,
            'apikey': systemKey,
            'allowedDownloads': allowedDownloads,
            'expiryTimestamp': expiry,
            'password': password,
            'originalPassword': originalPw
        },
    };
    try {
        const response = await fetch(apiUrl, requestOptions);
        if (!response.ok) {
            throw new Error(`Request failed with status: ${response.status}`);
        }
        const data = await response.json();
        return data;
    } catch (error) {
        console.error("Error in apiFilesModify:", error);
        throw error;
    }
}



async function apiFilesDelete(id) {
    const apiUrl = './api/files/delete';

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': systemKey,
            'id': id
        },
    };

    try {
        const response = await fetch(apiUrl, requestOptions);
        if (!response.ok) {
            throw new Error(`Request failed with status: ${response.status}`);
        }
    } catch (error) {
        console.error("Error in apiFilesDelete:", error);
        throw error;
    }
}


// users

async function apiUserModify(userId, permission, modifier) {
    const apiUrl = './api/user/modify';

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': systemKey,
            'userid': userId,
            'userpermission': permission,
            'permissionModifier': modifier

        },
    };

    try {
        const response = await fetch(apiUrl, requestOptions);
        if (!response.ok) {
            throw new Error(`Request failed with status: ${response.status}`);
        }
    } catch (error) {
        console.error("Error in apiUserModify:", error);
        throw error;
    }
}


async function apiUserDelete(id, deleteFiles) {
    const apiUrl = './api/user/delete';

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': systemKey,
            'userid': id,
            'deleteFiles': deleteFiles
        },
    };

    try {
        const response = await fetch(apiUrl, requestOptions);
        if (!response.ok) {
            throw new Error(`Request failed with status: ${response.status}`);
        }
    } catch (error) {
        console.error("Error in apiUserDelete:", error);
        throw error;
    }
}
