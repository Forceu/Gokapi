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

    const requestOptions = {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json',
            'apikey': systemKey,
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

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': systemKey,
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



// /chunk


async function apiChunkComplete(uuid, filename, filesize, realsize, contenttype, allowedDownloads, expiryDays, password, isE2E, nonblocking) {
    const apiUrl = './api/chunk/complete';

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': systemKey,
            'uuid': uuid,
            'filename': filename,
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


async function apiUserCreate(userName) {
    const apiUrl = './api/user/create';

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': systemKey,
            'username': userName

        },
    };

    try {
        const response = await fetch(apiUrl, requestOptions);
        if (!response.ok) {
        	if (response.status==409) {
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


async function apiUserChangeRank(userId, newRank) {
    const apiUrl = './api/user/changeRank';

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': systemKey,
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



async function apiUserResetPassword(id, generatePw) {
    const apiUrl = './api/user/resetPassword';

    const requestOptions = {
        method: 'POST', 
        headers: {
            'Content-Type': 'application/json',
            'apikey': systemKey,
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

    const requestOptions = {
        method: 'POST', 
        headers: {
            'Content-Type': 'application/json',
            'apikey': systemKey,
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


