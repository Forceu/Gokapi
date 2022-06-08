.. _usage:

=====
Usage
=====

Admin Menu
================


General
----------------

After you have started the Gokapi server, you can login using the your admin credentials by going to `http(s)://your.gokapi.url/admin``

There you can list and manage files and upload new files. You will also see three fields:

 - *Allowed downloads* lets you set how many times a file can be downloaded before it gets deleted
 - *Expiry in days* lets you set after how many days a file gets deleted latest
 - *Password* lets you set a password that a user needs to enter before downloading the file. Please note that the file on the storage server is not encrypted.

Uploading new files
---------------------

To upload, drag and drop a file, folder or multiple files to the Upload Zone. You can also directly paste an image from the clipboard. If you want to change the default expiry conditions, this has to be done before uploading. For each file an entry in the table will appear with a download link.

Identical files are deduplicated, which means if you upload a file twice, it will only be stored once.

Sharing files
---------------

Once you uploaded an file, you will see the options *Copy URL* and *Copy Hotlink*. By clicking on *Copy URL*, you copy the URL for the Download page to your clipboard. A user can then download the file from that page.

If a file does not require client-side decryption, you can also use the *Copy Hotlink* button. The hotlink URL is a direct link to the file and can for example be posted as an image on a forum or on a website. Each view counts as a download. Although Gokapi sets a Header to explicitly disallow caching, some browsers or external caches may still cache the image if they are not complient.


File deletion
---------------

Every hour Gokapi runs a cleanup routine which deletes all files from the storage that have been expired. If you click on the *Delete* button in the list, that file will be deleted from the disk immediately. AWS files are deleted after 24 hours, as of right now there is no proper way to find out if a download has been completed. 


API Menu
===============

In the API menu you can create API keys, which can be used for API access. Please refer to :ref:`api`.
