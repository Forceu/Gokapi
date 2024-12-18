.. _changelog:


Changelog
=========

Overview of all Changes
-----------------------

v1.9.6: 18 Dec 2024
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

* Add API call and GUI option to replace content of files (can be disabled with the env variable GOKAPI_DISABLE_REPLACE)
* Display error if encrypted download fails due to invalid SSL or CORS
* Better error handling for AWS setup check
* Fixed upload defaults being deleted when resetting e2e key
* Update download count in real time #206
* Fixed race condition that could lead to crash
* Change download count atomically to prevent race condition
* Renamed "Access Restriction" to indicate that authentication is disababled
* Make upload non blocking (#224), to prevent timouts after uploading large files
* Added API call /files/list/{id}
* Better handling for E2E errors
* Other minor changes
* Breaking Changes
   * API now returns 404 on invalid file IDs




v1.9.5: 08 Dec 2024
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

* Fixed a crash caused by an incorrectly upgraded database version #215

v1.9.4: 07 Dec 2024
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

* Retracted release

v1.9.3: 07 Dec 2024
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

* Fixed editing of API permissions or existing files not working, when using external authentication #210
* Fixed not showing an error message if file is larger than allowed file size #213
* Upload defaults are now saved locally instead of server-side #196
* Internal API key is now used for all API actions on the GUI
* Added API endpoint ``/auth/delete`` to delete API key
* Added parameter in ``/auth/create`` to include basic permissions
* Added warning in docker container, if data or config volume are not mounted
* Minor changes
* Breaking Changes
   * API: Session authentication has been removed, an API key is now required
   * API: When not adding a parameter for maximum downloads or expiry, the default values of 1 download or 14 days are used instead of previous used values for calls ``/files/add`` and ``/chunk/complete``

v1.9.2: 30 Sep 2024
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

* Added preview meta-data, enabling preview for services like WhatsApp
* Added hotlink support for avif and apng format
* Fixed headers not set when proxying S3 storage, resulting in incorrect filename and not forcing download

v1.9.1: 31 Jul 2024
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

* Fixed processing/uploading status not showing after upload #193 
* Fixed crash when OIDC returns nil for groups #198
* Fixed crash after running setup and changing encryption #197 
* Changed versioning of css/js files to prevent caching of old versions #195
* Other minor changes
* Breaking Changes
   * If you are using a custom theme, make sure that you change the CSS and JS filenames. Instead of e.g. main.min.css, the files are versioned now to include the version number in the filename, in this example the filename would be main.min.5.css


v1.9.0: 15 Jul 2024
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

* Fixed upload speeds being very low in some cases
* Fixed Docker image having the incorrect timezone
* Added Redis support. If you want to use Redis instead of SQLite, re-run the setup to change your database type. Refer to the `documentation <https://gokapi.readthedocs.io/en/stable/advanced.html#databases>`_ on how to migrate your data to a different database
* Database location can now be changed with the setup
* Fixed QR code not having decryption key when end-to-end encryption was enabled 
* Added option to display filenames in URL
* Added makefile for development
* Replaced SSE library with more efficient code
* Fixed ``go generate`` not working on Windows
* Gokapi version number will not be displayed on public pages anymore 
* Added ``windows/arm64`` target
* Breaking Changes
   * API: The output for the schema ``File`` has changed. The base URL was removed and now the complete URL for to download or hotlink the file is added. The additional key ``IncludeFilename`` indicates if the URLs contain the filename.
   * Configuration: Env variable ``GOKAPI_DB_NAME`` deprecated. On first start the database location will be saved as an URL string to the configuration file. For automatic deployment ``GOKAPI_DATABASE_URL`` can also be used


v1.8.4: 29 May 2024
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

* Gokapi runs as root in Docker container by default (this was changed in 1.8.3). To run it as unprivileged user, set environment variable DOCKER_NONROOT to true.
* Removed logging of errors when a user unexpectedly closed download or upload connection


v1.8.3: 27 May 2024
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

* Fixed Keycloak documentation. **Important:** If you have used the old example for configuration, please make sure that it is configure properly, as with the old example unauthorised access might have been possible! Documentation: Creating scopes for groups
* The binary will no longer be run as root in the Docker image. Breaking change: If you want to reconfigure Gokapi, the argument to pass to Docker is now a different one: Documentation
* If salts are empty, new ones will now be generated on startup. This is to aid automatic deployment
* A new admin password can be set with --deployment-password newPassword, but this should only be used for automatic deployment
* Env variable GOKAPI_LOG_STDOUT added, which also outputs all log entries to the terminal
* Display error message, if a reverse proxy does not allow file upload, or has been set to accept a too low file size
* Added header, so that nginx does not cache SSE
* Cloud storage file downloads can now be proxied through Gokapi, e.g. if the storage server is on an internal network
* Fixed a bug, where the option "Always save images locally" reverted back to default when re-running setup
* Updated documentation


v1.8.2: 20 Apr 2024
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

* Fixed that trailing slash was removed from OIDC provider URL: Thanks @JeroenoBoy
* S3 credentials are not shown in setup anymore, if they are provided through environment variables
* Added parameter to install Gokapi as a systemd service: Thanks @masoncfrancis
* Fixed typos: Thanks @Phaeton
* Updated Go version to 1.22


v1.8.1: 7 Feb 2024
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

* Reworked OIDC authentication, added support for Groups, use consent instead of select_account, better error handling
* Added wildcard support for OIDC groups and users
* Fixed crash on client timeout #125
* Added /auth/create API endpoint for creating API keys
* Minor changes and fixes


v1.8.0: 9 Dec 2023
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

* Parameters of already uploaded files can be edited now
* Added permission model for API tokens
* Added /auth/modify and /files/modify API endpoint
* Fixed "Powered by Gokapi" URL not clickable
* Fixed the ASCII logo #108 Thanks to @Kwonunn
* Improved UI
* Fixed minor bugs
* Updated dependencies
* Updated documentation
* Breaking Changes
   * Changed Database to Sqlite3
   * Dropped Windows 32bit support
   * Only 4,000 parallel requests that are writing to the database are supported now, any requests above that limit may be rejected. Up to 500,000 parallel reading requests were tested.
   * According to the documentation, the GOKAPI_DATA_DIR environment variable should be persistent, however that was not the case. Now the data directory that was set on first start will be used. If you were using GOKAPI_DATA_DIR after the first start, make sure that the data directory is the one found in your config file.
   * By default, IP addresses of clients downloading files are not saved anymore to comply with GDPR. This can be enabled by re-running the setup
   * Existing API keys will be granted all API permissions except MODIFY_API, therefore cannot use /auth/friendlyname without having the permission granted first
   * The undocumented GOKAPI_FILE_DB environment variable was removed
   * Removed optional application for reading database content


v1.7.2: 13 May 2023
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
* Added option to change the name in the setup
* The filename is now shown in the title for downloads
* SessionStorage is used instead of localStorage for e2e decryption
* Replaced expiry image with dynamic SVG


v1.7.1: 14 Apr 2023
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
* Fixed Gokapi not able to upload when running on a Windows system #95
* Improved Upload UI
* Added healthcheck for docker by @Jisagi in #89
* Fixed upload counter not updating after upload #92
* Fixed hotlink generation on files that required client-side decryption
* Replaced ``go:generate`` code with native Go
* Min Go version now 1.20
* Updated dependencies
* A lot of refactoring, minor changes
* Fixed background not loading in 1.7.0 (unpublished release) #101

v1.6.2: 14 Feb 2023
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
* Fixed timeout if a large file was uploaded to the cloud #81
* File overview is now sortable and searchable
* Added log viewer
* Updated Go to 1.20
* Other minor changes and fixes

v1.6.1: 17 Aug 2022
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

* Fixed setup throwing error 500 on docker installation


v1.6.0: 17 Aug 2022
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
* Use chunked uploads instead of single upload #68
* Add end-to-end encryption #71
* Fixed hotlink not being generated for uploads through API with unlimited storage time
* Added arm64 to Docker latest image
* Added API call to duplicate existing files
* Fixed bug where encrypted files could not be downloaded after rerunning setup
* Port selection is now disabled when running setup with docker
* Added timeout for AWS if endpoint is invalid
* Added flag to disable CORS check on startup
* Service worker for insecure connections is now hosted on Github
* "Noaws" version is not included as binary build anymore, but can be generated manually
* Breaking Changes
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


v1.5.2: 08 Jun 2022
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
* Added ARMv8 (ARM64) to Docker image
* Added option to always store images locally in order to support hotlink for encrypted files
* Fixed crash when remote files exist but system was changed to local files after running --reconfigure
* Added warning if incorrect CORS setting are set for AWS bucket
* Added button in setup to test AWS credentials
* Added more build infos to --version output
* Added download counter
* Added flags for port, config and data location, better flag usage overview
* Fixed that a file was reuploaded to AWS, even if it already existed
* Fixed error image for hotlinks not displaying if nosniff is enforced
* Fixed that two text files were created when pasting text
* Fixed docker image in documentation @emanuelduss

v1.5.1: 10 Mar 2022
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
* Fixed that selection of remote storage was not available during intitial setup
* Fixed that "bind to localhost" could be selected on docker image during initial setup
* Fixed that with Level 1 encryption remote files were encrypted as well
* If Gokapi is hosted under a https URL, the serviceworker for remote decryption is now included, which fixes that Firefox users with restrictive settings could not download encrypted files from remote storage
* Design improvements by @mraif13


v1.5.0: 08 Mar 2022
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
* Minimum version for upgrading is 1.3
* Encryption support for local and remote files
* Additional authentication methods: Header-Auth, OIDC and Reverse Proxy
* Option to allow unlimited downloads of files
* The configuration file has been partly replaced with a database. After the first start, the configuration file may be read-only
* A web-based setup instead of command line


v1.3.1: 03 Jul 2021
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
* Default upload limit is now 100GB and can be changed with environment variables on first start
* Fixed upload not working when using suburl on webserver for Gokapi
* Added log file
* Minor performance increase

v1.3.0: 17 May 2021
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

* Added cloudstorage support (AWS S3 / Backblaze B2)
* After changing password, all sessions will be logged out
* Fixed terminal input on Windows
* Added SSL support
* Documentation now hosted on ReadTheDocs

v1.2.0: 07 May 2021
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

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

v1.1.3: 07 Apr 2021
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

* Fixed bug where salts were not used anymore for password hashing
* Added hotlinking for image files
* Added logout button

v1.1.2: 03 Apr 2021
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

* Added support for env variables, major refactoring
* Configurations like length of the ID or salts can be changed with env variables now
* Fixed minor bugs, minor enhancements

v1.1.0: 18 Mar 2021
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

* Added option to password protect uploads
* Added ability to paste images into admin upload


v1.0.1: 12 Mar 2021
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

* Increased security of generated download IDs


v1.0: 12 Mar 2021
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

* First stable release of the program


Upgrading
-----------------------

Upgrading to 1.9
^^^^^^^^^^^^^^^^^^

* You need to update to Gokapi 1.8.4 before updating to Gokapi 1.9
* You might need to change permissions on the docker volumes, if you want the content to be readable by the host user. (Only applicable if you were running 1.8.3 before)
* If you have used the old Keycloak example for configuration, please make sure that it is configure properly, as with the old example unauthorised access might have been possible! `Documentation: Creating scopes for groups <https://gokapi.readthedocs.io/en/stable/examples.html#addding-a-scope-for-exposing-groups-optional>`_

Upgrading to 1.8
^^^^^^^^^^^^^^^^^^

* You need to update to Gokapi 1.7 before updating to Gokapi 1.8
* With this release, the old key-value database was changed to sqlite3. Please backup all Gokapi data before installing this release. On first start, the old database will be migrated and all users will be logged out. 

Upgrading to 1.5
^^^^^^^^^^^^^^^^^^

* You need to update to Gokapi 1.3 before updating to Gokapi 1.5
* After the upgrade the config file can be read-only
* Initial setup has to be done through a web interface now, setting Gokapi up through env variables is not possible anymore
* If you would like to use new features like a different authentication method, please run Gokapi with the parameter ``--reconfigure`` to open the setup  
* If you set the length of the file ID to 80 or more, you need to delete all files before running this update

Upgrading to 1.3
^^^^^^^^^^^^^^^^^^

* If you would like to use native SSL, please pass the environment variable ``GOKAPI_USE_SSL`` on first start after the update or manually edit the configuration file
* AWS S3 and Backblaze B2 can now be used instead of local storage! Please refer to the documentation on how to set it up.
