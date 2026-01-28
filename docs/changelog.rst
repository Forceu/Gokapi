.. _changelog:


Changelog
=========

Overview of all changes
-----------------------


v2.2.0 (not yet released)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* File Requests has been added, where you can request a file upload from other people
* Log Viewer has gotten a big overhaul and displays many other stats now
* It is now possible to use a custom favicon 
* Short-lived tokens are used instead of user API keys to improve security
* Browser timezone is used instead of server timezone for UI
* Added env variable to set a minium password length @masterbender 
* Downloads can be made from the UI without increasing the download counter
* gokapi-cli now supports downloads
* Add deprecation alerts @spaghetti-coder
* A lot of UI improvements
* Many small fixes and improvements


Breaking Changes
""""""""""""""""

* ``DOCKER_NONROOT`` has been deprecated in favour of ``docker --user``. See `documentation <https://gokapi.readthedocs.io/en/latest/setup.html#migration-from-docker-nonroot-to-docker-user>`__ on how to migrate
* API output for FileList has slightly changed
* Chunks must be at least 5MB in size, except the last chunk
* To delete logs, the user has to be an admin now


**Full Changelog**: https://github.com/Forceu/Gokapi/compare/v2.1.0...v2.2.0


v2.1.0 (2025-08-29)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Added a CLI tool that supports e2e encrypted uploads and folder uploads, see `documentation <https://gokapi.readthedocs.io/en/stable/advanced.html#cli-tool>`__ for installation and usage `#280 <https://github.com/Forceu/Gokapi/issues/280>`__
* Upgraded to Go 1.25 which might result in better performance on some systems
* Added docker-compose file
* Fixed crash after uploading an e2e encrypted file, forcing the user to refresh the webpage before uploading a new file `#283 <https://github.com/Forceu/Gokapi/issues/283>`__
* Fixed a bug where files with non-latin characters were not downloadable from AWS `#302 <https://github.com/Forceu/Gokapi/issues/302>`__ 
* Fixed a bug where e2e encrypted files with non-latin characters had a corrupted filename after downloading `#300 <https://github.com/Forceu/Gokapi/issues/300>`__
* Fixed bug where file was deleted after uploading through API if not supplying ``allowedDownloads`` or ``expiryDays`` in ``ChunkComplete`` `#282 <https://github.com/Forceu/Gokapi/issues/282>`__
* Fixed error message when username was less than 4 characters long `#268 <https://github.com/Forceu/Gokapi/issues/268>`__
* Fixed incorrect mouse pointer on share menu `#275 <https://github.com/Forceu/Gokapi/issues/275>`__
* Parallel uploads are now disabled, due to browser limit of 6 connections with HTTP1.1

**Full Changelog**: https://github.com/Forceu/Gokapi/compare/v2.0.1...v2.1.0


v2.0.1 (2025-06-08)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Fixed uploads failing for files with non-ASCII filenames `#269 <https://github.com/Forceu/Gokapi/issues/269>`__ 
* Fixed API documentation for API call ``/chunk/complete``
* Fixed rare edge case, where a file with a cancelled deletion was still deleted
* Filenames can now be base64-encoded in API call ``/chunk/complete``
* Added docker-compose file @SemvdH 


Upgrading
"""""""""

If you are upgrading from an older version than v2.0.0, please make sure to read the `v.2.0.0 upgrade notes <https://github.com/Forceu/Gokapi/releases/tag/v2.0.0>`__ first.

**Full Changelog**: https://github.com/Forceu/Gokapi/compare/v2.0.0...v2.0.1


v2.0.0 (2025-05-31)
^^^^^^^^^^^^^^^^^^^

.. warning::
     Make sure that you have a backup of all data. It is not possible to downgrade afterwards. 

This release adds user management and granular permission control. Some breaking changes are introduced, please make sure to read the section *Upgrading*. 


Security
""""""""

This releases fixes two XSS vulnerabilities (`CVE-2025-48494 <https://github.com/Forceu/Gokapi/security/advisories/GHSA-95rc-wc32-gm53>`__ and `CVE-2025-48495 <https://github.com/Forceu/Gokapi/security/advisories/GHSA-4xg4-54hm-9j77>`__). The vulnerabilities let authorised users execute Javascript with passive interaction - if you are using Gokapi as a single user, this does not impact you, otherwise we recommend updating your instance to v2.0.0.

Changelog
"""""""""

* Added support for multiple different users with granular permissions
* Added API endpoints to manage users
* Added API endpoint to delete logs, added more logging, added filtering and deletion of logs in UI
* Added feature to restore a deleted file from the UI (has to be restored within 5 seconds)
* Added API endpoint for restoring a file with a pending delete
* Added experimental hotlinking for videos with env var ``GOKAPI_ENABLE_HOTLINK_VIDEOS``
* Added a share button for mobile users and a button to share a URL via email
* Improved the UI
* Changed ``GOKAPI_LENGTH_ID``  to be non-permanent, added ``GOKAPI_LENGTH_HOTLINK_ID`` to change hotlink ID length `#251 <https://github.com/Forceu/Gokapi/issues/251>`__
* Changed hotlink URLs to be shorter (`#253 <https://github.com/Forceu/Gokapi/issues/253>`__) @lenisko 
* Changed headers for cache control to stop unwanted caching with cloudflare `#209 <https://github.com/Forceu/Gokapi/issues/209>`__
* Fixed email scope not being submitted `#234 <https://github.com/Forceu/Gokapi/issues/234>`__, fix always being redirected after successful OIDC login
* Fixed DuplicateFile setting hotlink on wrong file object (`#246 <https://github.com/Forceu/Gokapi/issues/246>`__)
* Fixed bug where picture files where not uploaded at all when encryption and cloud storage was active as well as ``SaveToLocal`` `#247 <https://github.com/Forceu/Gokapi/issues/247>`__
* Many other fixes and minor improvements @nilicule 

Upgrading
"""""""""

Upgrade path: **Requires v1.9.6 as base**, ``config.json`` must be writable

Upgrading when using OAuth2/OIDC authentication:
''''''''''''''''''''''''''''''''''''''''''''''''

 - A valid email must now be set for all users in the authentication backend
 - Authentication is now only done by email and can be restricted by user groups
 - Set the env variable ``GOKAPI_ADMIN_USER`` containing the email address of the super admin when upgrading 

Upgrading when using Header authentication
''''''''''''''''''''''''''''''''''''''''''

* If restricting the users by username, make sure that you remove any wildcards (*) for usernames in the setup before upgrading.
* Set the env variable ``GOKAPI_ADMIN_USER`` containing the email address of the super admin when upgrading

Upgrading when using no authentication
''''''''''''''''''''''''''''''''''''''

* If you are restricting access with a proxy, make sure that you block the following urls:

  * /admin
  * /apiKeys
  * /changePassword
  * /e2eInfo
  * /e2eSetup
  * /logs
  * /uploadChunk
  * /uploadStatus
  * /users
 

Upgrading when using custom templates or static content
'''''''''''''''''''''''''''''''''''''''''''''''''''''''

The previous way of replacing content has been removed and is now replaced with additive CSS and JS. If you want to change the layout (e.g. add your company logo or add/disable certain features), follow these steps:

1. Create a new folder named custom where your executable is. When using Docker, mount a new folder to /``app/custom/``. Any file in this directory will be publicly available in the sub-URL ``/custom/``.
2. To have custom CSS included, create a file in the folder named ``custom.css``. The CSS will be applied to all pages.
3. To have custom JavaScript included, create the file ``public.js`` for all public pages and/or ``admin.js`` for all admin-related pages. Please note that the ``admin.js`` will be readable to all users.
4. In order to prevent caching issues, you can version your files by creating the file ``version.txt`` with a version number.
5. Restart the server. If the folders exist, the server will now add the local files.

Optional: If you require further changes or want to embedded the changes permanently, you can clone the source code and then modify the templates in ``internal/webserver/web/templates``. Afterwards run ``make`` to build a new binary with these changes.

Breaking Changes
""""""""""""""""

Since v1.9 there have been a lot of changes to the API, please take note if you are using the API:

* A valid API key is now always required, API authentication by session is not possible anymore
* ``/chunk/complete`` and ``/files/duplicate`` now expect the parameters as header, instead of encoded url form
* Parameter ``apiKeyToModify`` has been renamed to ``targetKey`` for ``/auth/modify``, ``/auth/delete`` and ``/auth/friendlyname``
* If a user, api key or file is not found, but a plausible ID was submitted, error 404 instead of 400 is returned now
* Before v2.0, if a boolean parameter was required, it was always false if anything else then "true" was sent, now it raises an error if any other value than 1, t, true, 0, f, or false is supplied
* Some API calls might be restricted by user permissions now, consult the API documentation for more information
* API keys now have a public ID as well, which can also be used for ``/auth/modify``, ``/auth/delete`` and ``/auth/friendlyname`` as ``targetKey`` instead of the private ID
* When uploading a file through the API, defaults of 14 days, max 1 download and no password will be used, unless the respective parameters were passed. In v1.9, the previous values were used.


💙 **A huge thank you** to all our users, bug reporters, and contributors who made this release possible!

**Full Changelog**: https://github.com/Forceu/Gokapi/compare/v1.9.6...v2.0.0


v1.9.6 (2024-12-18)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Add API call and GUI option to replace content of files (can be disabled with the env variable ``GOKAPI_DISABLE_REPLACE``) `#128 <https://github.com/Forceu/Gokapi/issues/128>`__
* Display error if encrypted download fails due to invalid SSL or CORS
* Better error handling for AWS setup check
* Fixed upload defaults being deleted when resetting e2e key
* Update download count in real time `#206 <https://github.com/Forceu/Gokapi/issues/206>`__
* Fixed race condition that could lead to crash
* Change download count atomically to prevent race condition
* Renamed "Access Restriction" to indicate that authentication is disababled
* Make upload non blocking (`#224 <https://github.com/Forceu/Gokapi/issues/224>`__), to prevent timouts after uploading large files
* Added API call ``/files/list/{id}``
* Better handling for E2E errors
* Other minor changes

Breaking Changes
""""""""""""""""

* **API:** API now returns 404 on invalid file IDs

**Full Changelog**: https://github.com/Forceu/Gokapi/compare/v1.9.5...v1.9.6


v1.9.5 (2024-12-08)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Fixed a crash caused by an incorrectly upgraded database version `#215 <https://github.com/Forceu/Gokapi/issues/215>`__, `#216 <https://github.com/Forceu/Gokapi/issues/216>`__

**Full Changelog**: https://github.com/Forceu/Gokapi/compare/v1.9.4...v1.9.5


v1.9.3 (2024-12-07)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Fixed editing of API permissions or existing files not working, when using external authentication `#210 <https://github.com/Forceu/Gokapi/issues/210>`__ 
* Fixed not showing an error message if file is larger than allowed file size `#213 <https://github.com/Forceu/Gokapi/issues/213>`__
* Upload defaults are now saved locally instead of server-side `#196 <https://github.com/Forceu/Gokapi/issues/196>`__
* Internal API key is now used for all API actions on the GUI
* Added API endpoint ``/auth/delete`` to delete API key
* Added parameter in ``/auth/create`` to include basic permissions
* Added warning in docker container, if data or config volume are not mounted
* Minor changes

Breaking Changes
""""""""""""""""

* **API:** Session authentication has been removed, an API key is now required
* **API:** When not adding a parameter for maximum downloads or expiry, the default values of 1 download or 14 days are used instead of previous used values for calls ``/files/add`` and ``/chunk/complete``


**Full Changelog**: https://github.com/Forceu/Gokapi/compare/v1.9.2...v1.9.3


v1.9.2 (2024-09-30)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Added preview meta-data, enabling preview for services like WhatsApp
* Added hotlink support for avif and apng format
* Fixed headers not set when proxying S3 storage, resulting in incorrect filename and not forcing download `#199 <https://github.com/Forceu/Gokapi/issues/199>`__

Upgrading
"""""""""

* If running an older version than 1.9.2 please check the  `1.9.1 changelog <https://github.com/Forceu/Gokapi/releases/tag/v1.9.1>`__ for upgrading and breaking changes


**Full Changelog**: https://github.com/Forceu/Gokapi/compare/v1.9.1...v1.9.2


v1.9.1 (2024-07-31)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Fixed processing/uploading status not showing after upload `#193 <https://github.com/Forceu/Gokapi/issues/193>`__ 
* Fixed crash when OIDC returns nil for groups `#198 <https://github.com/Forceu/Gokapi/issues/198>`__
* Fixed crash after running setup and changing encryption `#197 <https://github.com/Forceu/Gokapi/issues/197>`__ 
* Changed versioning of css/js files to prevent caching of old versions `#195 <https://github.com/Forceu/Gokapi/issues/195>`__
* Other minor changes

Breaking changes
""""""""""""""""

If you are using a custom theme, make sure that you change the CSS and JS filenames. Instead of e.g. ``main.min.css``, the files are versioned now to include the version number in the filename, in this example the filename would be ``main.min.5.css``

Upgrading
"""""""""

* If running an older version than 1.9.0, please check the  `1.9.0 changelog <https://github.com/Forceu/Gokapi/releases/tag/v1.9.0>`__ for upgrading and breaking changes


**Full Changelog**: https://github.com/Forceu/Gokapi/compare/v1.9.0...v1.9.1


v1.9.0 (2024-07-15)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Fixed upload speeds being very low in some cases `#162 <https://github.com/Forceu/Gokapi/issues/162>`__
* Fixed Docker image having the incorrect timezone `#169 <https://github.com/Forceu/Gokapi/issues/169>`__
* Added Redis support. If you want to use Redis instead of SQLite, re-run the setup to change your database type. Refer to the `documentation <https://gokapi.readthedocs.io/en/stable/advanced.html#databases>`__ on how to migrate your data to a different database
* Database location can now be changed with the setup
* Fixed QR code not having decryption key when end-to-end encryption was enabled 
* Added option to display filenames in URL `#171 <https://github.com/Forceu/Gokapi/issues/171>`__
* Added makefile for development
* Replaced SSE library with more efficient code
* Fixed ``go generate`` not working on Windows, thanks @Kwonunn 
* Gokapi version number will not be displayed on public pages anymore 
* Added ``windows/arm64`` target

Breaking Changes
""""""""""""""""

* **API:** The output for the schema ``File`` has changed. The base URL was removed and now the complete URL for to download or hotlink the file is added. The additional key ``IncludeFilename`` indicates if the URLs contain the filename.
* **Configuration:** Env variable ``GOKAPI_DB_NAME`` deprecated. On first start the database location will be saved as an URL string to the configuration file. For automatic deployment ``GOKAPI_DATABASE_URL`` can also be used


Upgrading
"""""""""

* Configuration file needs to be writable
* If running an older version than 1.8.0, please upgrade to 1.8.4 first and check the  `1.8.0 changelog <https://github.com/Forceu/Gokapi/releases/tag/v1.8.0>`__ for upgrading and breaking changes


**Full Changelog**: https://github.com/Forceu/Gokapi/compare/v1.8.4...v1.9.0


v1.8.4 (2024-05-29)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Gokapi runs as root in Docker container by default (this was changed in 1.8.3). To run it as unprivileged user, set environment variable ``DOCKER_NONROOT`` to true.
* Removed logging of errors when a user unexpectedly closed download or upload connection

Upgrading
"""""""""

* You might need to change permissions on the docker volumes, if you want the content to be readable by the host user. (Only applicable if you were running 1.8.3 before)
* **Important**: If you have used the old Keycloak example for configuration, please make sure that it is configure properly, as with the old example unauthorised access might have been possible! `Documentation: Creating scopes for groups <https://gokapi.readthedocs.io/en/stable/examples.html#addding-a-scope-for-exposing-groups-optional>`__

If you are running a version <1.8.0, please see the `1.8.0 changelog <https://github.com/Forceu/Gokapi/releases/tag/v1.8.0>`__ for upgrading and breaking changes


**Full Changelog**: https://github.com/Forceu/Gokapi/compare/v1.8.3...v1.8.4


v1.8.3 (2024-05-27)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Fixed Keycloak documentation. **Important:** If you have used the old example for configuration, please make sure that it is configure properly, as with the old example unauthorised access might have been possible! `Documentation: Creating scopes for groups <https://gokapi.readthedocs.io/en/stable/examples.html#addding-a-scope-for-exposing-groups-optional>`__
* The binary will no longer be run as root in the Docker image. **Breaking change:** If you want to reconfigure Gokapi, the argument to pass to Docker is now a different one: `Documentation <https://gokapi.readthedocs.io/en/stable/setup.html#changing-configuration>`__
* If salts are empty, new ones will now be generated on startup. This is to aid `automatic deployment <https://gokapi.readthedocs.io/en/stable/advanced.html#automatic-deployment>`__
* A new admin password can be set with ``--deployment-password newPassword``, but this should only be used for automatic deployment
* Env variable ``GOKAPI_LOG_STDOUT`` added, which also outputs all log entries to the terminal
* Display error message, if a reverse proxy does not allow file upload, or has been set to accept a too low file size
* Added header, so that nginx does not cache SSE
* Cloud storage file downloads can now be proxied through Gokapi, e.g. if the storage server is on an internal network
* Fixed a bug, where the option "Always save images locally" reverted back to default when re-running setup
* Updated documentation

Upgrading
"""""""""

If you are running a version <1.8.0, please see the `1.8.0 changelog <https://github.com/Forceu/Gokapi/releases/tag/v1.8.0>`__ for upgrading and breaking changes


**Full Changelog**: https://github.com/Forceu/Gokapi/compare/v1.8.2...v1.8.3


v1.8.2 (2024-04-20)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Fixed that trailing slash was removed from OIDC provider URL: Thanks @JeroenoBoy 
* S3 credentials are not shown in setup anymore, if they are provided through environment variables
* Added parameter to install Gokapi as a systemd service: Thanks @masoncfrancis
* Fixed typos: Thanks @Phaeton 
* Updated Go version to 1.22

Upgrading
"""""""""

If you are running a version <1.8.0, please see the `1.8.0 changelog <https://github.com/Forceu/Gokapi/releases/tag/v1.8.0>`__ for upgrading and breaking changes

**Full Changelog**: https://github.com/Forceu/Gokapi/compare/v1.8.1...v1.8.2


v1.8.1 (2024-02-07)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Reworked OIDC authentication, added support for Groups, use consent instead of select_account, better error handling
* Added wildcard support for OIDC groups and users
* Fixed crash on client timeout `#125 <https://github.com/Forceu/Gokapi/issues/125>`__
* Added /auth/create API endpoint for creating API keys
* Minor changes and fixes

Upgrading
"""""""""

If you are running a version <1.8.0, please see the `1.8.0 changelog <https://github.com/Forceu/Gokapi/releases/tag/v1.8.0>`__ for upgrading and breaking changes



v1.8.0 (2023-12-09)
^^^^^^^^^^^^^^^^^^^

.. warning::
     Make sure that you have a backup of all data. It is not possible to downgrade afterwards.

With this release, the old key-value database was changed to sqlite3. Please backup all Gokapi data before installing this release. On first start, the old database will be migrated and all users will be logged out. If you experience any problems, please open an issue and let us know!


Changelog
"""""""""

* Parameters of already uploaded files can be edited now
* Added permission model for API tokens
* Added ``/auth/modify`` and ``/files/modify API`` endpoint
* Fixed "Powered by Gokapi" URL not clickable
* Fixed the ASCII logo `#108 <https://github.com/Forceu/Gokapi/issues/108>`__ Thanks to @Kwonunn 
* Improved UI
* Fixed minor bugs
* Updated dependencies
* Updated documentation

Breaking Changes
""""""""""""""""

* Dropped Windows 32bit support
* Only 4,000 parallel requests that are writing to the database are supported now, any requests above that limit may be rejected. Up to 500,000 parallel reading requests were tested.
* According to the documentation, the ``GOKAPI_DATA_DIR`` environment variable should be persistent, however that was not the case. Now the data directory that was set on first start will be used. If you were using ``GOKAPI_DATA_DIR`` after the first start, make sure that the data directory is the one found in your config file.
* By default, IP addresses of clients downloading files are not saved anymore to comply with GDPR. This can be enabled by re-running the setup
* Existing API keys will be granted all API permissions except ``MODIFY_API``, therefore cannot use ``/auth/friendlyname`` without having the permission granted first
* The undocumented ``GOKAPI_FILE_DB`` environment variable was removed
* Removed optional application for reading database content






v1.7.2 (2023-05-13)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""
* Added option to change the name in the setup
* The filename is now shown in the title for downloads
* SessionStorage is used instead of localStorage for e2e decryption
* Replaced expiry image with dynamic SVG


**Full Changelog**: https://github.com/Forceu/Gokapi/compare/v1.7.1...v1.7.2


v1.7.1 (2023-04-14)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""
* Fixed Gokapi not able to upload when running on a Windows system `#95 <https://github.com/Forceu/Gokapi/issues/95>`__ 
* Improved Upload UI
* Added healthcheck for docker by @Jisagi in https://github.com/Forceu/Gokapi/pull/89
* Fixed upload counter not updating after upload `#92 <https://github.com/Forceu/Gokapi/issues/92>`__ 
* Fixed hotlink generation on files that required client-side decryption
* Replaced go:generate code with native Go
* Min Go version now 1.20
* Updated dependencies
* A lot of refactoring, minor changes
* Fixed background not loading in 1.7.0 (unpublished release) `#101 <https://github.com/Forceu/Gokapi/issues/101>`__ 


**Full Changelog**: https://github.com/Forceu/Gokapi/compare/v1.6.2...v1.7.1


v1.6.2 (2023-02-13)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Fixed timeout if a large file was uploaded to the cloud `#81 <https://github.com/Forceu/Gokapi/issues/81>`__
* File overview is now sortable and searchable
* Added log viewer
* Updated Go to 1.20
* Other minor changes and fixes

**Full Changelog**: https://github.com/Forceu/Gokapi/compare/v1.6.1..v1.6.2


v1.6.1 (2022-08-17)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Fixed bug that prevented running setup with docker

**Full Changelog**: https://github.com/Forceu/Gokapi/compare/v1.6.0...v1.6.1


v1.6.0 (2022-08-17)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Use chunked uploads instead of single upload `#68 <https://github.com/Forceu/Gokapi/issues/68>`__
* Add end-to-end encryption `#71 <https://github.com/Forceu/Gokapi/issues/71>`__
* Fixed hotlink not being generated for uploads through API with unlimited storage time
* Added arm64 to Docker latest image
* Added API call to duplicate existing files
* Fixed bug where encrypted files could not be downloaded after rerunning setup 
* Port selection is now disabled when running setup with docker
* Added timeout for AWS if endpoint is invalid
* Added flag to disable CORS check on startup
* Service worker for insecure connections is now hosted on Github
* "Noaws" version is not included as binary build anymore, but can be generated manually

Breaking Changes
""""""""""""""""
* API output for fileuploads are less verbose and have changed parameters, please see updated OpenApi documentation
* If you disabled authentication, the following endpoints need to be secured:
  * /admin
  * /apiDelete
  * /apiKeys
  * /apiNew
  * /delete
  * /e2eInfo
  * /e2eSetup
  * /uploadChunk
  * /uploadComplete

Upgrading
"""""""""

* Minimum version for upgrading is 1.5
* Please make a backup before upgrading.
* Remove any custom templates or custom static files 
* Optionally run the server with the parameter ``--reconfigure`` to try out the new features.

Please report any issues you have with this release!

**Full Changelog**: https://github.com/Forceu/Gokapi/compare/v1.5.2...v1.6.0


v1.5.2 (2022-06-08)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Added ARMv8 (ARM64) to Docker image
* Added option to always store images locally in order to support hotlink for encrypted files
* Fixed crash when remote files exist but system was changed to local files after running ``--reconfigure``
* Added warning if incorrect CORS setting are set for AWS bucket
* Added button in setup to test AWS credentials
* Added more build infos to ``--version`` output
* Added download counter
* Added flags for port, config and data location, better flag usage overview
* Fixed that a file was reuploaded to AWS, even if it already existed
* Fixed error image for hotlinks not displaying if ``nosniff`` is enforced
* Fixed that two text files were created when pasting text
* Fixed docker image in documentation @emanuelduss

**Full Changelog**: https://github.com/Forceu/Gokapi/compare/v1.5.1...v1.5.2


v1.5.1 (2022-03-10)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Fixed that selection of remote storage was not available during intitial setup `#50 <https://github.com/Forceu/Gokapi/issues/50>`__ 
* Fixed that "bind to localhost" could be selected on docker image during initial setup
* Fixed that with Level 1 encryption remote files were encrypted as well
* If Gokapi is hosted under a https URL, the serviceworker for remote decryption is now included, which fixes that Firefox users with restrictive settings could not download encrypted files from remote storage `#49 <https://github.com/Forceu/Gokapi/issues/49>`__ 
* Design improvements by @mraif13 `#51 <https://github.com/Forceu/Gokapi/issues/51>`__


**Full Changelog**: https://github.com/Forceu/Gokapi/compare/v1.5.0...v1.5.1


v1.5.0 (2022-03-08)
^^^^^^^^^^^^^^^^^^^

**This release contains major changes, please read carefully**

Upgrading
"""""""""

* Minimum version for upgrading is 1.3
* Please make a backup before upgrading.
* Remove any custom templates or custom static files 
* Optionally run the server with the parameter ``--reconfigure`` to try out the new features.

Changelog
"""""""""

* Encryption support for local and remote files
* Additional authentication methods: Header-Auth, OIDC and Reverse Proxy
* Option to allow unlimited downloads of files
* The configuration file has been partly replaced with a database. After the first start, the configuration file may be read-only
* A web-based setup instead of command line

Please report any issues you have with this release! Especially if you are using the full encryption mode with S3, we are very happy about any feedback.

**Full Changelog**: https://github.com/Forceu/Gokapi/compare/v1.3.1...v1.5.0


v1.3.1 (2021-07-03)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Default upload limit is now 100GB and can be changed with environment variables on first start
* Fixed upload not working when using suburl on webserver for Gokapi
* Added log file
* Minor performance increase


v1.3.0 (2021-05-17)
^^^^^^^^^^^^^^^^^^^

Upgrading
"""""""""

* If you would like to use native SSL, please pass the environment variable ``GOKAPI_USE_SSL`` on first start after the update or manually edit the configuration file
* AWS S3 and Backblaze B2 can now be used instead of local storage! Please refer to the `documentation <https://gokapi.readthedocs.io/en/latest/setup.html#cloudstorage-setup>`__ on how to set it up.

Changelog
"""""""""

* Added cloudstorage support (AWS S3 / Backblaze B2)
* After changing password, all sessions will be logged out
* Fixed terminal input on Windows
* Added SSL support
* Documentation now hosted on ReadTheDocs

Different release versions
""""""""""""""""""""""""""

We now offer either a ``full`` and a ``noaws`` version. The ``full`` version contains open-source code from Amazon for connecting to their API, however also significantly increases the final size (around 35-40%). In the ``noaws`` version you can only store files on your local storage.


v1.2.0 (2021-05-07)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Fixed Docker images
* Added API
* Added header to prevent caching by browser / proxy
* Fixed upload timeout
* Added timeouts for server
* Added header to show download progress
* Prevent data races
* Cleanup routine does not delete files anymore while they are being downloaded
* Fixed that env ``LENGTH_ID`` was being ignored
* Show message if docker container is run on initial setup without ``-it``
* A lot of refactoring and minor improvements / bug fixes


v1.1.3 (2021-04-07)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Fixed bug where salts were not used anymore for password hashing
* Added hotlinking for image files
* Added logout button

Breaking Changes
""""""""""""""""

A developer version between v1.1.2  and v1.1.3 introduced a bug that prevented the usage of salts for hashing passwords! If you have only been using the regular releases, this notice does not apply to you.

If you created your admin account with a developer version of v1.1.2 or changed the password in a developer version of v1.1.2, you will need to run the following command: ``./gokapi --reset-pw``. You can enter the same password again. If you skip this step, you will be unable to login.

Files that have been password-protected with a developer version of v1.1.2 need to be uploaded again.


v1.1.2 (2021-04-03)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Added support for env variables, major refactoring
* Configurations like length of the ID or salts can be changed with env variables now
* Fixed minor bugs, minor enhancements 


v1.1.0 (2021-03-18)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Added option to password protect uploads
* Added ability to paste images into admin upload


v1.0.1 (2021-03-12)
^^^^^^^^^^^^^^^^^^^

Changelog
"""""""""

* Increased security of generated download IDs


v1.0 (2021-03-12)
^^^^^^^^^^^^^^^^^

Initial release


