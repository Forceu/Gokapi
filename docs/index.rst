.. _index:

===========================
Gokapi
===========================

Gokapi is a lightweight server to share files that expire after a set number of downloads or days. It is similar to the discontinued Firefox Send, with the difference that only authenticated users are allowed to upload files — anonymous visitors can only download.

This lets companies or individuals share files easily and have them removed automatically afterwards, saving disk space and keeping control over who can access the content.

Key features:

* **Expiring links** — files are deleted after a configurable number of downloads or after a set number of days
* **File Requests** — generate a link that lets external users upload files to your server
* **Multi-user support** — multiple accounts with granular per-user and per-API-key permissions
* **Deduplication** — identical files are stored only once
* **End-to-end encryption** — optional client-side encryption so even a compromised server cannot read file contents
* **S3-compatible cloud storage** — store files on AWS S3, Backblaze B2, or any S3-compatible provider instead of locally
* **CLI tool** — upload and download files directly from the command line
* **REST API** — full API for scripting and third-party integrations
* **Easy customisation** — change the look with plain CSS and JavaScript, no recompilation needed


Contents
========

.. toctree::
   :maxdepth: 2

   setup
   usage
   update
   advanced
   troubleshooting
   examples
   contributions
   changelog
