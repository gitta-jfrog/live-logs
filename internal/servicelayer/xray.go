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
	xrayVersionEndPoint = "api/v1/system/version"
	xrayMinVersionSupport = "3.18.0"
)

type xrayVersionData struct {
	Version string `json:"xray_version,omitempty"`
}

type XrayData struct {
	nodeId          string
	logFileName     string
	lastPageMarker  int64
	logsRefreshRate time.Duration
}

func (s *XrayData) GetConfig(ctx context.Context, serverId string) (*model.Config, error) {

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

func (s *XrayData) GetLogData(ctx context.Context, serverId string) (logData model.Data, err error) {
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

	resBody, err := clientlayer.SendGet(timeoutCtx, serverId, endpoint, s.nodeId,baseUrl,headers)

	if err != nil {
		return logData, err
	}

	if err := json.Unmarshal(resBody, &logData); err != nil {
		return logData, err
	}

	return logData, nil
}

func (s *XrayData) getConnectionDetails(serverId string)(url string, headers map[string]string,_ error){
	confDetails, err := cliCommands.GetConfig(serverId, false)
	if err != nil {
		return "",nil, err
	}
	url = confDetails.GetXrayUrl()
	accessToken := confDetails.GetAccessToken()
	if url == "" {
		return "",nil, fmt.Errorf("xray url is not found in serverId : %s, please make sure you using latest version of Jfrog CLI",serverId)
	}
	if accessToken == "" {
		return "",nil, fmt.Errorf("no access token found in serverId : %s, this is mandatory to connect to Xray product",serverId)
	}

	headers = make(map[string]string)
	headers["Authorization"] = "Bearer " + accessToken

	return url,headers, nil
}

func (s *XrayData) getVersion(ctx context.Context, serverId string) (string, error) {
	timeoutCtx, cancelTimeout := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancelTimeout()

	baseUrl, headers, err := s.getConnectionDetails(serverId)
	if err != nil {
		return "", err
	}
	resBody, err := clientlayer.SendGet(timeoutCtx, serverId, xrayVersionEndPoint,constants.EmptyNodeId, baseUrl, headers)

	if err != nil {
		return "", err
	}

	versionData := xrayVersionData{}
	err = json.Unmarshal(resBody, &versionData)
	if err != nil {
		return "", err
	}
	if versionData.Version == "" {
		return "", fmt.Errorf("could not retreive version information from Xray")
	}
	return strings.TrimSpace(versionData.Version), nil
}

func (s *XrayData) checkVersion(ctx context.Context, serverId string) error {
	currentVersion, err := s.getVersion(ctx, serverId)
	if err != nil {
		return err
	}
	if currentVersion == "" {
		return fmt.Errorf("api returned an empty version")
	}
	versionHelper := cliVersionHelper.NewVersion(xrayMinVersionSupport)

	if versionHelper.Compare(currentVersion) < 0 {
		return fmt.Errorf("found Xray version as %s, minimum supported version is %s", currentVersion, xrayMinVersionSupport)
	}
	return nil
}

func (s *XrayData) SetNodeId(nodeId string) {
	s.nodeId = nodeId
}

func (s *XrayData) SetLogFileName(logFileName string) {
	s.logFileName = logFileName
}

func (s *XrayData) SetLogsRefreshRate(logsRefreshRate time.Duration) {
	s.logsRefreshRate = logsRefreshRate
}

func (s *XrayData) SetLastPageMarker(pageMarker int64) {
	s.lastPageMarker = pageMarker
}

func (s *XrayData) GetLastPageMarker() int64 {
	return s.lastPageMarker
}

func (s *XrayData) GetNodeId() string {
	return s.nodeId
}

func (s *XrayData) GetLogFileName() string {
	return s.logFileName
}

func (s *XrayData) GetLogsRefreshRate() time.Duration {
	return s.logsRefreshRate
}
