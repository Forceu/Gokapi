package database

import (
	"github.com/forceu/gokapi/internal/configuration/database/dbabstraction"
	"github.com/forceu/gokapi/internal/models"
	// Required for sqlite driver
	_ "modernc.org/sqlite"
)

var db dbabstraction.Database

// Init connects to the database and creates the table structure, if necessary
func Init(config models.DbConnection) {
	db = dbabstraction.GetNew(config)
	err := db.Init(config)
	if err != nil {
		panic(err)
	}
}

// RunGarbageCollection runs the databases GC
func RunGarbageCollection() {
	db.RunGarbageCollection()
}

// Upgrade migrates the DB to a new Gokapi version, if required
func Upgrade(currentVersion int) {
	db.Upgrade(currentVersion)
}

// Close the database connection
func Close() {
	db.Close()
}

// Api Key Section

// GetAllApiKeys returns a map with all API keys
func GetAllApiKeys() map[string]models.ApiKey {
	return db.GetAllApiKeys()
}

// GetApiKey returns a models.ApiKey if valid or false if the ID is not valid
func GetApiKey(id string) (models.ApiKey, bool) {
	return db.GetApiKey(id)
}

// SaveApiKey saves the API key to the database
func SaveApiKey(apikey models.ApiKey) {
	db.SaveApiKey(apikey)
}

// UpdateTimeApiKey writes the content of LastUsage to the database
func UpdateTimeApiKey(apikey models.ApiKey) {
	db.UpdateTimeApiKey(apikey)
}

// DeleteApiKey deletes an API key with the given ID
func DeleteApiKey(id string) {
	db.DeleteApiKey(id)
}

// E2E Section

// SaveEnd2EndInfo stores the encrypted e2e info
func SaveEnd2EndInfo(info models.E2EInfoEncrypted) {
	db.SaveEnd2EndInfo(info)
}

// GetEnd2EndInfo retrieves the encrypted e2e info
func GetEnd2EndInfo() models.E2EInfoEncrypted {
	return db.GetEnd2EndInfo()
}

// DeleteEnd2EndInfo resets the encrypted e2e info
func DeleteEnd2EndInfo() {
	db.DeleteEnd2EndInfo()
}

// Hotlink Section

// GetHotlink returns the id of the file associated or false if not found
func GetHotlink(id string) (string, bool) {
	return db.GetHotlink(id)
}

// GetAllHotlinks returns an array with all hotlink ids
func GetAllHotlinks() []string {
	return db.GetAllHotlinks()
}

// SaveHotlink stores the hotlink associated with the file in the database
func SaveHotlink(file models.File) {
	db.SaveHotlink(file)
}

// DeleteHotlink deletes a hotlink with the given hotlink ID
func DeleteHotlink(id string) {
	db.DeleteHotlink(id)
}

// Metadata Section

// GetAllMetadata returns a map of all available files
func GetAllMetadata() map[string]models.File {
	return db.GetAllMetadata()
}

// GetAllMetaDataIds returns all Ids that contain metadata
func GetAllMetaDataIds() []string {
	return db.GetAllMetaDataIds()
}

// GetMetaDataById returns a models.File from the ID passed or false if the id is not valid
func GetMetaDataById(id string) (models.File, bool) {
	return db.GetMetaDataById(id)
}

// SaveMetaData stores the metadata of a file to the disk
func SaveMetaData(file models.File) {
	db.SaveMetaData(file)
}

// DeleteMetaData deletes information about a file
func DeleteMetaData(id string) {
	db.DeleteMetaData(id)
}

// Session Section

// GetSession returns the session with the given ID or false if not a valid ID
func GetSession(id string) (models.Session, bool) {
	return db.GetSession(id)
}

// SaveSession stores the given session. After the expiry passed, it will be deleted automatically
func SaveSession(id string, session models.Session) {
	db.SaveSession(id, session)
}

// DeleteSession deletes a session with the given ID
func DeleteSession(id string) {
	db.DeleteSession(id)
}

// DeleteAllSessions logs all users out
func DeleteAllSessions() {
	db.DeleteAllSessions()
}

// Upload Defaults Section

// GetUploadDefaults returns the last used setting for amount of downloads allowed, last expiry in days and
// a password for the file
func GetUploadDefaults() models.LastUploadValues {
	values, ok := db.GetUploadDefaults()
	if ok {
		return values
	}
	defaultValues := models.LastUploadValues{
		Downloads:         1,
		TimeExpiry:        14,
		Password:          "",
		UnlimitedDownload: false,
		UnlimitedTime:     false,
	}
	return defaultValues
}

// SaveUploadDefaults saves the last used setting for an upload
func SaveUploadDefaults(values models.LastUploadValues) {
	db.SaveUploadDefaults(values)
}

// Upload Status Section

// GetAllUploadStatus returns all UploadStatus values from the past 24 hours
func GetAllUploadStatus() []models.UploadStatus {
	return db.GetAllUploadStatus()
}

// GetUploadStatus returns a models.UploadStatus from the ID passed or false if the id is not valid
func GetUploadStatus(id string) (models.UploadStatus, bool) {
	return db.GetUploadStatus(id)
}

// SaveUploadStatus stores the upload status of a new file for 24 hours
func SaveUploadStatus(status models.UploadStatus) {
	db.SaveUploadStatus(status)
}
