.. _usage:

=====
Usage
=====

Upload Menu
================


General
----------------

After you have started the Gokapi server, you can login using the your credentials by going to `http(s)://your.gokapi.url/admin``

There you can list and manage files and upload new files. You will also see three fields:

 - *Allowed downloads* lets you set how many times a file can be downloaded before it gets deleted
 - *Expiry in days* lets you set after how many days a file gets deleted latest
 - *Password* lets you set a password that a user needs to enter before downloading the file. Please note that the file on the storage server is not encrypted.

Uploading new files
---------------------

To upload, drag and drop a file, folder or multiple files to the Upload Zone. You can also directly paste an image or text from the clipboard. If you want to change the default expiry conditions, this has to be done before uploading. For each file an entry in the table will appear with a download link.

Identical files are deduplicated, which means if you upload a file twice, it will only be stored once.

Sharing files
---------------

Once you uploaded an file, you will see a button with the options *Copy URL* and *Copy Hotlink*. By clicking on *Copy URL*, you copy the URL for the Download page to your clipboard. A user can then download the file from that page.

If a file does not require client-side decryption, you can also use the *Copy Hotlink* button. The hotlink URL is a direct link to the file and can for example be posted as an image on a forum or on a website. Each view counts as a download. Although Gokapi sets a Header to explicitly disallow caching, some browsers or external caches may still cache the image if they are not compliant.

The second button lets you share the regular URL easily. If you are accessing Gokapi with a mobile device, a tap on the button will open your device's share menu. Otherwise you can click on the drop down element and select to either share the link via email or generate a QR code.

Downloading files
------------------

The upload menu has a button which lets you download a file without increasing the download counter. You can also click on the file ID to go to the regular download page, which increases the counter.

Editing files
---------------

By clicking on the edit button, you can change limits like the maximum download count or replace the file with the contents of a different uploaded file.

File deletion
---------------

Every hour Gokapi runs a cleanup routine which deletes all files from the storage that have been expired. If you click on the *Delete* button in the list, that file will be deleted from the disk immediately. Unproxied AWS files are deleted after 24 hours, as of right now there is no proper way to find out if a download has been completed. 


File Request Menu
===================


General
----------------

The File Requests page allows you to create secure, invitation-only upload links. These links enable external users to send files directly to your server without needing an account.


.. note::
   **Security Note:** If End-to-End Encryption is enabled globally, please note that **File Requests bypass this**. All files uploaded through the upload request page will be in plain text. This does only affect servers with end-to-end encryption, regular file encryption is still in place.

Dashboard
---------------------------

The main dashboard provides a summary of all active and expired file requests.

* **Name**: The friendly name of the request. Clicking this link opens the public upload page in a new tab.
* **Uploaded Files**: Displays the number of files currently received.

    * **+X**: Indicates "active" uploads currently in progress.
    * **X / Max**: Shows the current count against a set file limit.
* **Total Size**: The combined storage footprint of all files in that request.
* **Last Upload**: The date and time the most recent file was added.
* **Expiry**: When the link will stop accepting new uploads.
* **Actions**: Quick tools to manage, download, or delete the request.

Managing Files
---------------------------

Each row in the table can be expanded to view and manage individual files.

Viewing Files
^^^^^^^^^^^^^^
If a request has files, a *chevron (down arrow)* icon will appear next to the file count. Clicking this will expand a list showing:

* Individual file names.
* File sizes and upload dates.
* Direct download buttons for single files.

Downloading Content
^^^^^^^^^^^^^^^^^^^^
You can download files in two ways:

1.  **Single File**: Click the file name or the download icon within the expanded list.
2.  **Batch Download**: Click the download icon in the *Actions* column. If multiple files exist, the system will automatically package them into a ``.zip`` archive.

Creating and Editing Requests
---------------------------------

To create a new request, click the *Plus* icon at the top right. To modify an existing one, click the *Pencil* icon.

.. list-table::
   :widths: 20 80
   :header-rows: 1

   * - Field
     - Description
   * - **Title**
     - A friendly name to identify the request (e.g., "Project Assets").
   * - **Max Files**
     - Limit how many files users can upload to this link.
   * - **Max Size**
     - Set a maximum total size (in MB) for the entire request.
   * - **Expiry**
     - Set a date after which the link will no longer function.
   * - **Notes**
     - Public notes that are shown on the upload page


.. note::
   By default, non-admin users are limited to requesting up to 100 files, with a maximum size of 10 GB per file. To modify these limits or disable them entirely, set the environment variables ``GOKAPI_MAX_FILES_GUESTUPLOAD`` and ``GOKAPI_MAX_SIZE_GUESTUPLOAD`` to your desired values. See :ref:`availenvvar` for details.



Sharing and Deletion
--------------------

Sharing the Request
^^^^^^^^^^^^^^^^^^^^^^
1.  Locate the request in the table.
2.  Click the *Copy (Clipboard)* icon.
3.  A notification will confirm the URL is copied. You can now paste this into an email or chat.

Deleting Requests
^^^^^^^^^^^^^^^^^^^
To remove a request, click the *Trash* icon. 

.. warning::
   Deleting a File Request is permanent. This action also deletes all associated files currently stored on the server. This cannot be undone.




User Management
=================

The **Users** page provides administrators with tools to create accounts, manage permissions, and oversee user activity. This interface ensures you can delegate responsibilities while maintaining system security.

General
----------------------------

The user table displays a high-level summary of all accounts on the server:

* **User**: The display name or username of the account.
* **Group**: The account type (e.g., "Admin" or "User").
* **Last Online**: A timestamp indicating the last time the user logged into the system.
* **Uploads**: The total number of files currently owned by that user.
* **Permissions**: A quick-view grid of icons representing specific rights.
* **Actions**: Tools to reset passwords, promote/demote ranks, or delete accounts.

Managing Permissions
---------------------

Permissions are granular and can be toggled by clicking the icons in the **Permissions** column. 

.. list-table:: 
   :widths: 30 60
   :header-rows: 1

   * - Name
     - Description
   * - Create File Requests
     - Allows the user to generate external upload links.
   * - Replace Own Uploads
     - Allows the user to overwrite files they previously uploaded.
   * - List Other Uploads
     - Grant visibility to files uploaded by other system users.
   * - Edit Other Uploads
     - Allows editing files owned by others.
   * - Delete Other Uploads
     - Allows permanent removal of files owned by other users.
   * - Manage Logs
     - Grants access to view and clear system activity logs.
   * - Manage Users
     - Grants access to this User Management page.
   * - Manage API Keys
     - Allows management of API keys of belonging to any user

.. note::
   Permissions for the Super Admin and your own account cannot be modified from this screen to prevent accidental lockouts.

User Account Actions
---------------------

Adding a New User
^^^^^^^^^^^^^^^^^^

1. Click the *Plus (+)* icon at the top right of the Users card.
2. Enter a unique username.
3. The user will be created with default permissions and will need a password assigned or reset.

Resetting Passwords
^^^^^^^^^^^^^^^^^^^^
If using internal authentication, click the *Key* icon:

* **Force Reset**: The user must choose a new password the next time they log in.
* **Generate Random**: The system provides a temporary password. You can copy it to your clipboard to give to the user.


User Ranks
------------------
There are three different user ranks:

* **Super Admin**: A single person with all access which cannot be modified by other users.
* **Admin**: Has all rights by default. Is able to delete system logs and can change file owners. Can create upload requests with unlimited files and file size.
* **User**: Has less rights by default.



Changing User Rank
^^^^^^^^^^^^^^^^^^^
Use the *Chevron Up/Down* icons to change a user's group:

* **Promote**: Upgrades a standard User to an Admin.
* **Demote**: Downgrades an Admin to a standard User.



Deleting Users
--------------
Click the **Trash** icon to remove an account. 

.. warning::
   When deleting a user, you will be asked if you also want to **permanently delete all files** uploaded by them. If unchecked, the files will remain on the server and change the ownership to the user who initiated the deletion.








API Menu
===============

General
--------------


The API Keys page allows you to generate and manage credentials for programmatic access to the server. These keys are used to authenticate scripts, third-party applications, or CLI tools.

.. note::
   For technical implementation details and endpoint definitions, please refer to the integrated API Documentation and the section :ref:`api`
   
API Keys
---------------------

The API table provides a summary of all active credentials:

* **Name**: A descriptive label for the key (e.g., ``Internal Upload Tool``). You can click the name at any time to rename it.
* **API Key**: A redacted version of the key for security. When a new key is created, the full string will be displayed once - ensure you copy it immediately.
* **Last Used**: The timestamp of the most recent request made using this key.
* **Permissions**: A grid of icons representing what the key is authorized to do.
* **User**: (Admin only) Displays which system user owns the specific API key.

Managing Key Permissions
--------------------------

Permissions for API keys are granular. You can enable or disable a right by clicking its corresponding icon. 

.. list-table::
   :widths: 30 60
   :header-rows: 1

   * - Name
     - Description
   * - List Uploads
     - View a list of files currently on the server.
   * - Upload
     - Permission to push new files to the server.
   * - Edit Uploads
     - Modify metadata of existing files.
   * - Delete Uploads
     - Permanently remove files via the API.
   * - Replace
     - Overwrite existing files with new versions.
   * - Download
     - Retrieve file contents programmatically without increasing the download counter
   * - File Requests
     - Create and manage external "File Request" links.
   * - Manage Users
     - Create or modify user accounts via API calls.
   * - Manage Keys
     - Use this key to create or delete other API keys.

.. note::
   Some permissions may appear greyed out. This happens if the user who owns the key does not have that specific permission assigned to their account. An API key cannot grant more power than its owner possesses.

Key Operations
----------------

Creating a New Key
^^^^^^^^^^^^^^^^^^^

1. Click the *Plus (+)* icon in the top right corner.
2. A new key will be generated.
3. **Copy the key immediately.** For security reasons, the full key cannot be displayed again once you navigate away from the page.

Deleting a Key
^^^^^^^^^^^^^^^^^^^
To revoke access immediately, click the *Trash* icon in the Actions column. Any application using this key will instantly receive an ``Unauthorized`` error.



System Logs
==========================

The **Log File** page provides a view of system activity, security events, and file operations.

Filtering Logs
-----------------

To help you find specific information quickly, you can use the *Log Filter* dropdown menu. Selecting a category will parse the log file and display only the relevant lines.

.. list-table:: 
   :widths: 25 75
   :header-rows: 1

   * - Category
     - Description
   * - **Warning**
     - Non-critical errors or alerts that may require attention.
   * - **Auth**
     - Login attempts, password resets, and permission changes.
   * - **Download**
     - Records of files being accessed or downloaded by users/guests.
   * - **Upload**
     - New file creations and completed upload sessions.
   * - **Edit**
     - Metadata changes, file renames, and setting updates.
   * - **Info**
     - General operational status messages.

Log Maintenance and Cleanup
------------------------------

Over time, log files can become quite large. Administrators have access to the *Delete Logs* utility to manage storage and keep the logs readable.

.. note::
   The log deletion tool is restricted to users with *Administrator* privileges. For standard users, this menu will be disabled.

Retention Options
-----------------

You can clear logs based on their age using the following presets:

* **Older than 2/7/14/30 days**: Retains recent history while purging stale data.
* **Delete all logs**: Completely clears the log file.

.. warning::
   Log deletion is a permanent action. Once logs are cleared, the data cannot be recovered via the web interface. It is recommended to keep at least 7 days of logs for security auditing purposes.


