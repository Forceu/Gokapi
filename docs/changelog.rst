.. _changelog:


Changelog
=========

Overview of all Changes
-----------------------


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

Upgrading to 1.5
^^^^^^^^^^^^^^^^^^

* You need to update to Gokapi 1.3 before updating to Gokapi 1.5
* After the upgrade the config file can be read-only
* Initial setup has to be done through a web interface now, setting Gokapi up through env variables is not possible anymore
* If you would like to use new features like a different authentication method, please run Gokapi with the paramter ``--reconfigure`` to open the setup  
* If you set the length of the file ID to 80 or more, you need to delete all files before running this update

Upgrading to 1.3
^^^^^^^^^^^^^^^^^^

* If you would like to use native SSL, please pass the environment variable ``GOKAPI_USE_SSL`` on first start after the update or manually edit the configuration file
* AWS S3 and Backblaze B2 can now be used instead of local storage! Please refer to the documentation on how to set it up.
