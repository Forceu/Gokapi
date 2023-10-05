# TODO

## Feature: Guest Uploading
- [ ] New admin page to manage guest uploading
  - [ ] Generate guest token with properties
    - [ ] Max no. of files
    - [ ] Upload quota
  - [ ] View active tokens + their uploads
    - [ ] Remaining files and quota
    - [ ] Get links for guest uploads
    - [ ] Also allow Updating/Deleting
- [ ] New guest upload page /guestupload
  - [ ] First, enter token
  - [ ] Display remaining no. of files and quota
  - [ ] Upload like admin
    - [ ] Only accept if file isn't too big

## Tasks
- [x] Add guest tokens to database
- [x] Create admin panel page to make tokens
- [x] Create page to upload
  - [ ] Create matching error page
- [x] Add webserver endpoint for uploading
- [x] Normalize the various View structs
- [ ] Add the options back in
  - [ ] Update the options on the admin page
  - [ ] Update the options on the guest page
  - [ ] Actually implement the new options
- [ ] Un-break E2E encryption
- [ ] Check for possible token circumvention
