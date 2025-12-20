// This file contains JS code to connect to the API
// All files named admin_*.js will be merged together and minimised by calling
// go generate ./...


const storedTokens = new Map();

async function getToken(permission, forceRenewal) {
    const apiUrl = './auth/token';

    if (!forceRenewal) {
        if (!storedTokens.has(permission)) {
            return getToken(permission, true);
        }
        let token = storedTokens.get(permission);
        if (token.expiry - (Date.now() / 1000) < 60) {
            return getToken(permission, true);
        }
        return token.key;
    }
    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'permission': permission

        },
    };
    try {
        const response = await fetch(apiUrl, requestOptions);
        if (!response.ok) {
            throw new Error(`Request failed with status: ${response.status}`);
        }
        const data = await response.json();
        if (!data.hasOwnProperty("key")) {
            throw new Error(`Invalid response when trying to get token`);
        }
        storedTokens.set(permission, {
            key: data.key,
            expiry: data.expiry
        });
        return data.key;
    } catch (error) {
        console.error("Error in getToken:", error);
        throw error;
    }
}

// /auth

async function apiAuthModify(apiKey, permission, modifier) {
    const apiUrl = './api/auth/modify';
    const reqPerm = 'PERM_API_MOD';
    
    let token;

    try {
        token = await getToken(reqPerm, false);
    } catch (error) {
        console.error("Unable to gain permission token:", error);
        throw error;
    }

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': token,
            'targetKey': apiKey,
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
    const reqPerm = 'PERM_API_MOD';
    
    let token;

    try {
        token = await getToken(reqPerm, false);
    } catch (error) {
        console.error("Unable to gain permission token:", error);
        throw error;
    }

    const requestOptions = {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json',
            'apikey': token,
            'targetKey': apiKey,
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
    const reqPerm = 'PERM_API_MOD';
    
    let token;

    try {
        token = await getToken(reqPerm, false);
    } catch (error) {
        console.error("Unable to gain permission token:", error);
        throw error;
    }

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': token,
            'targetKey': apiKey,
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
    const reqPerm = 'PERM_API_MOD';
    
    let token;

    try {
        token = await getToken(reqPerm, false);
    } catch (error) {
        console.error("Unable to gain permission token:", error);
        throw error;
    }

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': token,
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



// /chunk


async function apiChunkComplete(uuid, filename, filesize, realsize, contenttype, allowedDownloads, expiryDays, password, isE2E, nonblocking) {
    const apiUrl = './api/chunk/complete';
    const reqPerm = 'PERM_UPLOAD';
    
    let token;

    try {
        token = await getToken(reqPerm, false);
    } catch (error) {
        console.error("Unable to gain permission token:", error);
        throw error;
    }

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': token,
            'uuid': uuid,
            'filename': 'base64:' + Base64.encode(filename),
            'filesize': filesize,
            'realsize': realsize,
            'contenttype': contenttype,
            'allowedDownloads': allowedDownloads,
            'expiryDays': expiryDays,
            'password': password,
            'isE2E': isE2E,
            'nonblocking': nonblocking
        },
    };

    try {
        const response = await fetch(apiUrl, requestOptions);
        if (!response.ok) {
            let errorMessage;

            // Attempt to parse JSON, fallback to text if parsing fails
            try {
                const errorResponse = await response.json();
                errorMessage = errorResponse.ErrorMessage || `Request failed with status: ${response.status}`;
            } catch {
                // Handle non-JSON error
                const errorText = await response.text();
                errorMessage = errorText || `Request failed with status: ${response.status}`;
            }
            throw new Error(errorMessage);
        }
        const data = await response.json();
        return data;
    } catch (error) {
        console.error("Error in apiChunkComplete:", error);
        throw error;
    }
}


// /files


async function apiFilesReplace(id, newId) {
    const apiUrl = './api/files/replace';
    const reqPerm = 'PERM_REPLACE';
    
    let token;

    try {
        token = await getToken(reqPerm, false);
    } catch (error) {
        console.error("Unable to gain permission token:", error);
        throw error;
    }

    const requestOptions = {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json',
            'id': id,
            'apikey': token,
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
    const reqPerm = 'PERM_VIEW';
    
    let token;

    try {
        token = await getToken(reqPerm, false);
    } catch (error) {
        console.error("Unable to gain permission token:", error);
        throw error;
    }
    
    const requestOptions = {
        method: 'GET',
        headers: {
            'Content-Type': 'application/json',
            'apikey': token,

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
    const reqPerm = 'PERM_EDIT';
    
    let token;

    try {
        token = await getToken(reqPerm, false);
    } catch (error) {
        console.error("Unable to gain permission token:", error);
        throw error;
    }

    const requestOptions = {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json',
            'id': id,
            'apikey': token,
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



async function apiFilesDelete(id, delay) {
    const apiUrl = './api/files/delete';
    const reqPerm = 'PERM_DELETE';
    
    let token;

    try {
        token = await getToken(reqPerm, false);
    } catch (error) {
        console.error("Unable to gain permission token:", error);
        throw error;
    }

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': token,
            'id': id,
            'delay': delay
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


async function apiFilesRestore(id) {
    const apiUrl = './api/files/restore';
    const reqPerm = 'PERM_DELETE';
    
    let token;

    try {
        token = await getToken(reqPerm, false);
    } catch (error) {
        console.error("Unable to gain permission token:", error);
        throw error;
    }

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': token,
            'id': id
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
        console.error("Error in apiFilesRestore:", error);
        throw error;
    }
}



// users


async function apiUserCreate(userName) {
    const apiUrl = './api/user/create';
    const reqPerm = 'PERM_MANAGE_USERS';
    
    let token;

    try {
        token = await getToken(reqPerm, false);
    } catch (error) {
        console.error("Unable to gain permission token:", error);
        throw error;
    }

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': token,
            'username': userName

        },
    };

    try {
        const response = await fetch(apiUrl, requestOptions);
        if (!response.ok) {
            if (response.status == 409) {
                throw new Error("duplicate");
            }
            throw new Error(`Request failed with status: ${response.status}`);
        }
        const data = await response.json();
        return data;
    } catch (error) {
        console.error("Error in apiUserModify:", error);
        throw error;
    }
}


async function apiUserModify(userId, permission, modifier) {
    const apiUrl = './api/user/modify';
    const reqPerm = 'PERM_MANAGE_USERS';
    
    let token;

    try {
        token = await getToken(reqPerm, false);
    } catch (error) {
        console.error("Unable to gain permission token:", error);
        throw error;
    }

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': token,
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


async function apiUserChangeRank(userId, newRank) {
    const apiUrl = './api/user/changeRank';
    const reqPerm = 'PERM_MANAGE_USERS';
    
    let token;

    try {
        token = await getToken(reqPerm, false);
    } catch (error) {
        console.error("Unable to gain permission token:", error);
        throw error;
    }

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': token,
            'userid': userId,
            'newRank': newRank

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
    const reqPerm = 'PERM_MANAGE_USERS';
    
    let token;

    try {
        token = await getToken(reqPerm, false);
    } catch (error) {
        console.error("Unable to gain permission token:", error);
        throw error;
    }

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': token,
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



async function apiUserResetPassword(id, generatePw) {
    const apiUrl = './api/user/resetPassword';
    const reqPerm = 'PERM_MANAGE_USERS';
    
    let token;

    try {
        token = await getToken(reqPerm, false);
    } catch (error) {
        console.error("Unable to gain permission token:", error);
        throw error;
    }

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': token,
            'userid': id,
            'generateNewPassword': generatePw
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
        console.error("Error in apiUserResetPassword:", error);
        throw error;
    }
}



async function apiLogsDelete(timestamp) {
    const apiUrl = './api/logs/delete';
    const reqPerm = 'PERM_MANAGE_LOGS';
    
    let token;

    try {
        token = await getToken(reqPerm, false);
    } catch (error) {
        console.error("Unable to gain permission token:", error);
        throw error;
    }

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': token,
            'timestamp': timestamp
        },
    };

    try {
        const response = await fetch(apiUrl, requestOptions);
        if (!response.ok) {
            throw new Error(`Request failed with status: ${response.status}`);
        }
    } catch (error) {
        console.error("Error in apiLogsDelete:", error);
        throw error;
    }
}

// E2E


async function apiE2eGet() {
    const apiUrl = './api/e2e/get';
    const reqPerm = 'PERM_UPLOAD';
    
    let token;

    try {
        token = await getToken(reqPerm, false);
    } catch (error) {
        console.error("Unable to gain permission token:", error);
        throw error;
    }

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': token
        },
    };

    try {
        const response = await fetch(apiUrl, requestOptions);
        if (!response.ok) {
            throw new Error(`Request failed with status: ${response.status}`);
        }
        return await response.text();
        // return await response.json();
    } catch (error) {
        console.error("Error in apiE2eGet:", error);
        throw error;
    }
}


async function apiE2eStore(content) {
    const apiUrl = './api/e2e/set';
    const reqPerm = 'PERM_UPLOAD';
    
    let token;

    try {
        token = await getToken(reqPerm, false);
    } catch (error) {
        console.error("Unable to gain permission token:", error);
        throw error;
    }

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': token
        },
        body: JSON.stringify({
            'content': content
        }),
    };

    try {
        const response = await fetch(apiUrl, requestOptions);
        if (!response.ok) {
            throw new Error(`Request failed with status: ${response.status}`);
        }
    } catch (error) {
        console.error("Error in apiE2eStore:", error);
        throw error;
    }
}

async function apiURequestDelete(id) {
    const apiUrl = './api/uploadrequest/delete';

    const requestOptions = {
        method: 'DELETE',
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
        console.error("Error in apiURequestDelete:", error);
        throw error;
    }
}
