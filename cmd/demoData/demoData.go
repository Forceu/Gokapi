package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/database"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/logging/serverStats"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/storage/filerequest"
)

type FileEntry struct {
	Name      string
	SizeStr   string
	SizeBytes int64
}

func main() {
	fmt.Println("WARNING: This will delete all data in the database!")
	fmt.Println("Press enter to continue...")
	_, _ = fmt.Scanln()
	configuration.Load()
	configuration.ConnectDatabase()

	file, _ := os.OpenFile(configuration.GetEnvironment().DataDir+"/xxxxxxxxxxxxxxxxxxxxxxxxxxxx", os.O_RDONLY|os.O_CREATE, 0644)
	file.Close()
	deleteAllData()
	addTraffic()
	addUsers()
	addFiles(4, "")
	addApiKeys()
	addFileRequests()
}

func addTraffic() {
	serverStats.AddTraffic(2818730232000)
}

func deleteAllData() {
	serverStats.ClearTraffic()
	for _, file := range database.GetAllMetadata() {
		database.DeleteMetaData(file.Id)
	}
	for _, apikey := range database.GetAllApiKeys() {
		database.DeleteApiKey(apikey.Id)
	}
	for _, fileRequest := range database.GetAllFileRequests() {
		filerequest.Delete(fileRequest)
	}
}

func addFileRequests() {
	for _, name := range filerequestNames {
		request := filerequest.New(models.User{Id: 2})
		request.Name = name
		apiKey := newApiKey(2)
		request.ApiKey = apiKey.Id
		apiKey.UploadRequestId = request.Id
		database.SaveApiKey(apiKey)
		database.SaveFileRequest(request)
		addFiles(rand.Intn(10), request.Id)
	}
}

func newApiKey(userId int) models.ApiKey {
	return models.ApiKey{
		Id:           helper.GenerateRandomString(30),
		PublicId:     helper.GenerateRandomString(60),
		FriendlyName: "New API Key",
		Permissions:  models.ApiPermission(rand.Intn(511)),
		UserId:       userId,
	}
}

func addApiKeys() {
	for _, apiName := range apiNames {
		apikey := models.ApiKey{
			Id:           helper.GenerateRandomString(30),
			PublicId:     helper.GenerateRandomString(60),
			FriendlyName: apiName,
			Permissions:  models.ApiPermission(rand.Intn(511)),
			UserId:       rand.Intn(4),
		}
		database.SaveApiKey(apikey)
	}
}

func addUsers() {
	users := []models.User{{
		Id:            1,
		Name:          "Alice Admin",
		Permissions:   models.UserPermissionAll,
		UserLevel:     models.UserLevelSuperAdmin,
		LastOnline:    time.Now().Add(time.Duration(rand.Intn(24*60)) * -time.Minute).Unix(),
		Password:      "fewfeefefwefweffwe",
		ResetPassword: false,
	}, {
		Id:            2,
		Name:          "Bob Uploader",
		Permissions:   models.UserPermissionAll,
		UserLevel:     models.UserLevelAdmin,
		LastOnline:    time.Now().Add(time.Duration(rand.Intn(24*60)) * -time.Minute).Unix(),
		Password:      "fewfeefefwefweffwe",
		ResetPassword: false,
	}, {
		Id:            3,
		Name:          "Charlie Viewer",
		Permissions:   models.UserPermissionNone,
		UserLevel:     models.UserLevelUser,
		LastOnline:    time.Now().Add(time.Duration(rand.Intn(24*60)) * -time.Minute).Unix(),
		Password:      "fewfeefefwefweffwe",
		ResetPassword: false,
	}, {
		Id:            4,
		Name:          "Dora Developer",
		Permissions:   models.UserPermission(rand.Intn(511)),
		UserLevel:     models.UserLevelUser,
		LastOnline:    time.Now().Add(time.Duration(rand.Intn(24*60)) * -time.Minute).Unix(),
		Password:      "fewfeefefwefweffwe",
		ResetPassword: false,
	}}
	for _, user := range users {
		database.SaveUser(user, false)
	}
}

func addFiles(count int, fileRequestId string) {
	for i := 0; i < count; i++ {
		exampleFile := fileNames[rand.Intn(len(fileNames))]
		newFile := models.File{
			Id:                      helper.GenerateRandomString(10),
			Name:                    exampleFile.Name,
			Size:                    exampleFile.SizeStr,
			SHA1:                    "xxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			PasswordHash:            "",
			HotlinkId:               "",
			ContentType:             "",
			AwsBucket:               "",
			UploadRequestId:         fileRequestId,
			ExpireAt:                0,
			PendingDeletion:         0,
			SizeBytes:               exampleFile.SizeBytes,
			UploadDate:              time.Now().Add(time.Duration(-rand.Intn(24*6000)) * time.Minute).Unix(),
			DownloadsRemaining:      0,
			DownloadCount:           rand.Intn(10000),
			UserId:                  rand.Intn(4) + 1,
			Encryption:              models.EncryptionInfo{},
			UnlimitedDownloads:      true,
			UnlimitedTime:           true,
			InternalRedisEncryption: nil,
		}
		if rand.Intn(2) == 0 {
			newFile.UnlimitedDownloads = false
			newFile.DownloadsRemaining = rand.Intn(1000)
		}
		if rand.Intn(2) == 0 {
			newFile.UnlimitedTime = false
			newFile.ExpireAt = time.Now().Add(time.Hour * time.Duration(rand.Intn(24*100))).Unix()
		}
		database.SaveMetaData(newFile)
	}
}

var apiNames = []string{
	"Main Upload Service",
	"CI/CD Pipeline",
	"Backup Automation",
	"Monitoring Agent",
	"Internal Tools",
	"Customer Upload Portal",
	"Mobile App Backend",
	"File Processing Worker",
	"Admin Script Access",
	"Temporary Migration Key",
}

var fileNames = []FileEntry{
	{"Quarterly Report Q1 2024.pdf", "3.2 MB", 3_355_443},
	{"Quarterly Report Q2 2024.pdf", "3.8 MB", 3_981_312},
	{"Annual Report 2023.pdf", "12.4 MB", 12_996_736},
	{"Company Presentation.pptx", "18.6 MB", 19_503_744},
	{"Marketing Assets.zip", "245.0 MB", 256_901_120},
	{"Product Photos.zip", "512.3 MB", 537_919_488},
	{"Invoice 2024 01.pdf", "182 KB", 186_368},
	{"Invoice 2024 02.pdf", "176 KB", 180_224},
	{"Invoice 2024 03.pdf", "190 KB", 194_560},
	{"Signed Contract ACME.pdf", "1.1 MB", 1_152_512},
	{"HR Policies Handbook.pdf", "4.7 MB", 4_926_208},
	{"Employee List.xlsx", "980 KB", 1_003_520},
	{"Payroll January.xlsx", "1.4 MB", 1_467_776},
	{"Payroll February.xlsx", "1.5 MB", 1_572_864},
	{"Server Backup 2024-01-01.tar.gz", "1.8 GB", 1_932_735_488},
	{"Server Backup 2024-02-01.tar.gz", "1.9 GB", 2_038_075_776},
	{"Database Dump.sql", "620 MB", 650_117_120},
	{"Website Assets.tar", "340 MB", 356_515_840},
	{"Design Mockups.fig", "96.5 MB", 101_082_624},
	{"Logo Pack.svg.zip", "14.2 MB", 14_886_016},
	{"Event Photos January.zip", "1.2 GB", 1_288_490_112},
	{"Event Photos February.zip", "980 MB", 1_027_254_400},
	{"Training Video Onboarding.mp4", "420 MB", 440_401_920},
	{"Webinar Recording.mp4", "860 MB", 901_775_360},
	{"Meeting Recording January.mp3", "85 MB", 89_270_400},
	{"Meeting Recording February.mp3", "92 MB", 96_544_512},
	{"Customer Feedback.csv", "640 KB", 655_360},
	{"Bug Report List.xlsx", "1.2 MB", 1_258_291},
	{"Security Audit Report.pdf", "6.8 MB", 7_126_528},
	{"PenTest Results.pdf", "9.3 MB", 9_746_688},
	{"Source Code Release v1.2.zip", "72 MB", 75_497_472},
	{"Source Code Release v1.3.zip", "78 MB", 81_777_408},
	{"Mobile App Build.apk", "110 MB", 115_343_360},
	{"iOS TestFlight Build.ipa", "240 MB", 251_658_240},
	{"UX Research Notes.docx", "2.1 MB", 2_199_552},
	{"Wireframes Final.sketch", "48 MB", 50_331_648},
	{"Sprint Plan March.xlsx", "720 KB", 737_280},
	{"Sprint Plan April.xlsx", "740 KB", 757_760},
	{"Legal Disclosure.pdf", "520 KB", 532_480},
	{"NDA Template.docx", "310 KB", 317_440},
	{"Customer List Internal.xlsx", "1.9 MB", 1_991_680},
	{"API Documentation.pdf", "5.4 MB", 5_662_848},
	{"System Architecture Diagram.png", "3.6 MB", 3_769_728},
	{"Release Notes v2.0.txt", "64 KB", 65_536},
	{"Monitoring Logs January.zip", "410 MB", 430_872_960},
	{"Monitoring Logs February.zip", "455 MB", 477_626_880},
	{"Compliance Evidence.zip", "880 MB", 922_746_880},
	{"Migration Plan.docx", "1.6 MB", 1_677_824},
}

var filerequestNames = []string{
	"Customer Document Submission",
	"Support File Upload",
	"External Partner Upload",
	"Secure Contract Exchange",
}
