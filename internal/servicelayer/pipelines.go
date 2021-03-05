package servicelayer

import (
	"context"
	"encoding/json"
	"fmt"
	cliCommands "github.com/jfrog/jfrog-cli-core/common/commands"
	cliVersionHelper "github.com/jfrog/jfrog-client-go/utils/version"
	"github.com/jfrog/live-logs/internal/clientlayer"
	"github.com/jfrog/live-logs/internal/constants"
	"github.com/jfrog/live-logs/internal/model"
	"strings"
	"time"
)

const (
	pipelinesVersionEndPoint = "api/v1/system/info"
	pipelinesMinVersionSupport = "1.13.0"
)

type pipelinesVersionData struct {
	Version string `json:"version,omitempty"`
}

type PipelinesData struct {
	nodeId          string
	logFileName     string
	lastPageMarker  int64
	logsRefreshRate time.Duration
}

func (s *PipelinesData) GetConfig(ctx context.Context, serverId string) (*model.Config, error) {

	err := s.checkVersion(ctx, serverId)
	if err != nil {
		return nil, err
	}

	timeoutCtx, cancelTimeout := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancelTimeout()
	baseUrl, headers, err := s.getConnectionDetails(serverId)
	if err != nil {
		return nil, err
	}
	resBody, err := clientlayer.SendGet(timeoutCtx, serverId, constants.ConfigEndpoint,constants.EmptyNodeId,baseUrl,headers)

	if err != nil {
		return nil, err
	}

	logConfig := model.Config{}
	err = json.Unmarshal(resBody, &logConfig)
	if err != nil {
		return nil, err
	}
	if len(logConfig.LogFileNames) == 0 {
		return nil, fmt.Errorf("no log file names were found")
	}
	if len(logConfig.Nodes) == 0 {
		return nil, fmt.Errorf("no node names were found")
	}
	return &logConfig, nil
}

func (s *PipelinesData) GetLogData(ctx context.Context, serverId string) (logData model.Data, err error) {
	if s.nodeId == "" {
		return logData, fmt.Errorf("node id must be set")
	}
	if s.logFileName == "" {
		return logData, fmt.Errorf("log file name must be set")
	}

	err = s.checkVersion(ctx, serverId)
	if err != nil {
		return logData, err
	}

	timeoutCtx, cancelTimeout := context.WithTimeout(ctx, defaultLogRequestTimeout)
	defer cancelTimeout()

	var endpoint string
	endpoint = fmt.Sprintf("%s?file_size=%d&id=%s", constants.DataEndpoint, s.lastPageMarker, s.logFileName)
	baseUrl, headers, err := s.getConnectionDetails(serverId)
	if err != nil {
		return logData, err
	}
	resBody, err := clientlayer.SendGet(timeoutCtx, serverId, endpoint, s.nodeId, baseUrl,headers)

	if err != nil {
		return logData, err
	}

	if err := json.Unmarshal(resBody, &logData); err != nil {
		return logData, err
	}

	return logData, nil
}

func (s *PipelinesData) getConnectionDetails(serverId string)(url string, headers map[string]string,_ error){
	confDetails, err := cliCommands.GetConfig(serverId, false)
	if err != nil {
		return "",nil, err
	}
	url = confDetails.GetPipelinesUrl()
	accessToken := confDetails.GetAccessToken()
	if url == "" {
		return "",nil, fmt.Errorf("pipelines url is not found in serverId : %s, please make sure you using latest version of Jfrog CLI",serverId)
	}
	if accessToken == "" {
		return "",nil, fmt.Errorf("no access token found in serverId : %s, this is mandatory to connect to Pipelines product",serverId)
	}

	headers = make(map[string]string)
	headers["Authorization"] = "Bearer " + accessToken

	return url,headers, nil
}

func (s *PipelinesData) getVersion(ctx context.Context, serverId string) (string, error) {
	timeoutCtx, cancelTimeout := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancelTimeout()

	baseUrl, headers, err := s.getConnectionDetails(serverId)
	if err != nil {
		return "", err
	}
	resBody, err := clientlayer.SendGet(timeoutCtx, serverId, pipelinesVersionEndPoint,constants.EmptyNodeId, baseUrl, headers)

	if err != nil {
		return "", err
	}

	versionData := pipelinesVersionData{}
	err = json.Unmarshal(resBody, &versionData)
	if err != nil {
		return "", err
	}
	if versionData.Version == "" {
		return "", fmt.Errorf("could not retreive version information from Pipelines")
	}

	return strings.TrimSpace(versionData.Version), nil
}

func (s *PipelinesData) checkVersion(ctx context.Context, serverId string) error {
	currentVersion, err := s.getVersion(ctx, serverId)
	if err != nil {
		return err
	}
	if currentVersion == "" {
		return fmt.Errorf("api returned an empty version")
	}
	versionHelper := cliVersionHelper.NewVersion(pipelinesMinVersionSupport)

	if versionHelper.Compare(currentVersion) < 0 {
		return fmt.Errorf("found JFrog Pipelines version as %s, minimum supported version is %s", currentVersion, pipelinesMinVersionSupport)
	}
	return nil
}

func (s *PipelinesData) SetNodeId(nodeId string) {
	s.nodeId = nodeId
}

func (s *PipelinesData) SetLogFileName(logFileName string) {
	s.logFileName = logFileName
}

func (s *PipelinesData) SetLogsRefreshRate(logsRefreshRate time.Duration) {
	s.logsRefreshRate = logsRefreshRate
}

func (s *PipelinesData) SetLastPageMarker(pageMarker int64) {
	s.lastPageMarker = pageMarker
}

func (s *PipelinesData) GetLastPageMarker() int64 {
	return s.lastPageMarker
}

func (s *PipelinesData) GetNodeId() string {
	return s.nodeId
}

func (s *PipelinesData) GetLogFileName() string {
	return s.logFileName
}

func (s *PipelinesData) GetLogsRefreshRate() time.Duration {
	return s.logsRefreshRate
}
