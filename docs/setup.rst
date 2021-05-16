.. _setup:

=====
Setup
=====

There are two different ways to setup Gokapi: Either a bare metal approach or docker.

Also there are two different versions: *Stable* indicates that you are using the latest release which should work without any bugs. *Unstable* is the latest developer version, which might include more features, but could also contain bugs.


**************
Installation
**************

Bare Metal
^^^^^^^^^^^^

Stable version
"""""""""""""""""
`Download the project <https://github.com/Forceu/gokapi/releases/latest>`_ and copy the executable into a new folder with write permissions.

Unstable version
"""""""""""""""""

Only recommended if you have expierence with the command line. Go 1.16+ needs to be installed.

Create a new folder and in this folder execute 
::

 git clone https://github.com/Forceu/Gokapi.git .
 go build Gokapi/cmd/gokapi

This will compile the source code and create an executable from the latest code.


Docker
^^^^^^^

To download, run the following command, and replace YOURTAG with either *latest* (stable) or *latest-dev* (unstable).
::

  docker pull f0rc3/barcodebuddy-docker:YOURTAG

Most of the time, you will need the *latest* tag. 

If you don't want to download the prebuilt image, you can find the Dockerfile on the `Github project page <https://github.com/Forceu/gokapi>`_. 



**************
First Start
**************

During the first start you will be asked several questions for the inital setup. To automate the setup, all questions can be preset with environment variables as well, see :ref:`envvar`


Starting Gokapi
^^^^^^^^^^^^^^^^

Bare Metal
""""""""""

To start Gokapi, execute the binary with your command line or by double clicking.


Docker
""""""""""

To start the container, run the following command: ::

 docker run -it -v gokapi-data:/app/data -v gokapi-config:/app/config -p 127.0.0.1:53842:53842 f0rc3/gokapi:latest

Please note the ``-it`` flag, which is needed if you are not populating all setup questions with environment variables. 

With the argument ``-p 127.0.0.1:53842:53842`` the service will only be accessible from the machine it is running on. In most usecases you will use a reverse proxy for SSL - if you want to make the service available to other computers in the network without a reverse proxy, replace the argument with ``-p 53842:53842``. Please note, unless you select SSL during the setup, the traffic will not be encrypted that way and data like passwords and transferred files can easily be read by 3rd parties!


Initial Setup
^^^^^^^^^^^^^^^

During the first start, a new configuration file will be created. You will be asked questions for all required values that have not been populated with environment variables, see :ref:`envvar`

The following values are required:

+-----------------------------+-------------------------------------------------------------------------------------------------------------+----------------------------------------+-----------------------------------+
| Question                    | Expected Entry                                                                                              | Expected format                        | Default                           |
+=============================+=============================================================================================================+========================================+===================================+
| Username                    | Username used for admin login (only user that can upload files)                                             | string, min 4 characters               |                                   |
+-----------------------------+-------------------------------------------------------------------------------------------------------------+----------------------------------------+-----------------------------------+
| Password                    | Password used for admin login                                                                               | string, min 6 characters               |                                   |
+-----------------------------+-------------------------------------------------------------------------------------------------------------+----------------------------------------+-----------------------------------+
| Server Port                 | The port Gokapi listens on                                                                                  | int, 0-65353, >1024 recommended        | 53842                             |
+-----------------------------+-------------------------------------------------------------------------------------------------------------+----------------------------------------+-----------------------------------+
| External                    | The URL that will be used for generating Gokapi download links.                                             | url, starting with http:// or https:// | http://127.0.0.1:53842/           |
| Server URL                  | Use an URL that users from an external network can use to reach Gokapi.                                     |                                        |                                   |
|                             | For testing purposes you can use the default value                                                          |                                        |                                   |
+-----------------------------+-------------------------------------------------------------------------------------------------------------+----------------------------------------+-----------------------------------+
| URL that the index          | By default Gokapi redirects to another URL instead of showing a generic page if no download link was passed | url, starting with http:// or https:// | https://github.com/Forceu/Gokapi/ |
| gets redirected to          |                                                                                                             |                                        |                                   |
+-----------------------------+-------------------------------------------------------------------------------------------------------------+----------------------------------------+-----------------------------------+
| Bind port to localhost only | If bound to localhost, Gokapi can only be accessed from the machine it runs on.                             | "y"/"yes" or "n"/"no"                  | Yes                               |
|                             | Recommended to set to "yes" if you use a reverse proxy or run Gokapi for testing purposes.                  |                                        |                                   |
+-----------------------------+-------------------------------------------------------------------------------------------------------------+----------------------------------------+-----------------------------------+
| Use SSL                     | If set to "yes", Gokapi will serve the content on the port with HTTPS.                                      | "y"/"yes" or "n"/"no"                  | No                                |
|                             | If no valid certificate is present in the config folder, a new one will be generated.                       |                                        |                                   |
+-----------------------------+-------------------------------------------------------------------------------------------------------------+----------------------------------------+-----------------------------------+


.. _cloudstorage:

********************
Cloudstorage Setup
********************

By default Gokapi uses local storage. You can also use external cloud storage providers for file storage. Please note that currently no native encryption is available for Gokapi, therefore all files will be stored in plain text on the cloud server.


AWS S3 / Backblaze B2
^^^^^^^^^^^^^^^^^^^^^^

Provider setup
""""""""""""""""""

It is highly recommended to create a new bucket for Gokapi and set it to "private", so that no file can be downloaded externally. For each download request Gokapi will create a public URL that is only valid for a couple of seconds, so that the file can be downloaded from the external server directly instead of routing it through the local server.

You then need to create an app key with read-/write-access to this bucket.

Local setup
""""""""""""

It is recommended to pass the credentials as environment variables to Gokapi, see :ref:`envvar`. They can however also be loaded from a configuration file. You can find an example file `here <https://github.com/Forceu/Gokapi/blob/master/example/cloudconfig.yml>`_. Modify the values and copy it as ``cloudconfig.yml`` into your ``config`` folder.

The following values can be parsed:

+-----------+-----------------------------------------------+-----------------------+-----------------------------------+
| Key       | Description                                   | Required              | Example                           |
+===========+===============================================+=======================+===================================+
| Bucket    | Name of the bucket in use                     | yes                   | gokapi                            |
+-----------+-----------------------------------------------+-----------------------+-----------------------------------+
| Region    | Name of the region                            | yes                   | eu-central-1                      |
+-----------+-----------------------------------------------+-----------------------+-----------------------------------+
| KeyId     | Name of the API key                           | yes                   | keyname123456789                  |
+-----------+-----------------------------------------------+-----------------------+-----------------------------------+
| KeySecret | Value of the API key secret                   | yes                   | verysecret123                     |
+-----------+-----------------------------------------------+-----------------------+-----------------------------------+
| Endpoint  | Endpoint to use. Leave blank if using AWS S3. | only for Backblaze B2 | s3.eu-central-001.backblazeb2.com |
+-----------+-----------------------------------------------+-----------------------+-----------------------------------+


