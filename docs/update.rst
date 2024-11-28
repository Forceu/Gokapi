.. _update:

======================
Updating Gokapi
======================

***************
Docker
***************

To update, run the following command:
::

  docker pull f0rc3/gokapi:YOURTAG

Then stop the running container and follow the same steps as in SETUP. All userdata will be preserved, as it is saved to the ``gokapi-data`` and ``gokapi-data`` volume (``-v`` argument during creation) 

*******************
Native deployment
*******************

Stable version
==============

To update, download the latest release and unzip it to the directory that contains the old version. Overwrite any existing files.


Unstable version
=================

To update, execute the command ``git pull`` and then rebuild the binary with ``go build Gokapi/cmd/gokapi``.
