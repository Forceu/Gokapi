.. _changelog:


Changelog
=========

Overview of all Changes
-----------------------


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

See TODO for upgrade instructions
