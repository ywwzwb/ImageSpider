package interfaces

const DataCheckerServiceID ServiceID = "DataChecker"

type IDataCheckerService interface {
	StartChecking(sourceID string)
}
