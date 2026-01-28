package interfaces

const FileIntegrityCheckerServiceID ServiceID = "FileIntegrityChecker"

type IFileIntegrityCheckerService interface {
	StartScanning(sourceID string)
}
