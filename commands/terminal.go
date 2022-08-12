package commands

import (
	"context"
	"fmt"
	cliCommands "github.com/jfrog/jfrog-cli-core/v2/common/commands"
	"github.com/jfrog/live-logs/internal"
	"github.com/jfrog/live-logs/internal/constants"
	"github.com/jfrog/live-logs/internal/model"
	"github.com/jfrog/live-logs/internal/util"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var PromptForAnyKey = util.PromptAndWaitForAnyKey
var PromptSelectMenu = util.RunInteractiveMenu
var CliServerIds = cliCommands.GetAllServerIds

func ListenForTermination(cancelCtx context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGABRT)
	go func() {
		<-c
		cancelCtx()
		fmt.Println("\r- Terminating")
		os.Exit(0)
	}()
}

func ConfigInteractive(ctx context.Context, liveLog livelog.LiveLogs) error {
	selectedProductId, err := selectProductId()
	if err != nil {
		return err
	}

	liveLog.SetProductId(selectedProductId)
	selectedCliServerId, err := selectCliServerId()
	if err != nil {
		return err
	}

	liveLog.SetServiceId(selectedCliServerId)
	nonInteractiveMessage := constants.NonIntCmdDisplayPrefix + " \n\t jfrog live-logs config " +
		selectedProductId + " " +
		selectedCliServerId
	PromptForAnyKey(nonInteractiveMessage)
	return liveLog.DisplayConfig(ctx)
}

func selectCliServerId() (string, error) {
	serverIds := CliServerIds()
	return PromptSelectMenu("Select JFrog CLI server id", "Available server IDs", serverIds)
}

func selectProductId() (string, error) {
	productIds := util.FetchAllProductIds()
	return PromptSelectMenu("Select JFrog CLI product id", "Available product IDs", productIds)
}

func LogInteractiveMenu(ctx context.Context, isStreaming bool, liveLog livelog.LiveLogs) error {
	selectedProductId, err := selectProductId()
	if err != nil {
		return err
	}
	liveLog.SetProductId(selectedProductId)

	selectedCliServerId, err := selectCliServerId()
	if err != nil {
		return err
	}
	liveLog.SetServiceId(selectedCliServerId)

	nodeId, logName, logsRefreshRate, err := selectLogDetails(ctx, liveLog)
	if err != nil {
		return err
	}
	liveLog.SetLogsRefreshRate(logsRefreshRate)

	cmdDisplayPostfix := ""
	if isStreaming {
		cmdDisplayPostfix = " -" + constants.TailFlag + cmdDisplayPostfix
	}

	nonInteractiveMessage := constants.NonIntCmdDisplayPrefix + " \n\t jfrog live-logs logs " +
		selectedProductId + " " +
		selectedCliServerId + " " +
		nodeId + " " +
		logName + cmdDisplayPostfix
	PromptForAnyKey(nonInteractiveMessage)
	return liveLog.PrintLogs(ctx, nodeId, logName, isStreaming)
}

func selectLogDetails(ctx context.Context, liveLog livelog.LiveLogs) (selectedNodeID string, selectedLogName string, logsRefreshRate time.Duration, err error) {
	var srvConfig *model.Config
	srvConfig, err = liveLog.GetConfigData(ctx, liveLog.GetProductId(), liveLog.GetServiceId())
	if err != nil {
		return
	}

	logsRefreshRate = util.MillisToDuration(srvConfig.RefreshRateMillis)
	selectedNodeID, err = PromptSelectMenu("Select Node Id", "Available Node Ids", srvConfig.Nodes)
	selectedLogName, err = PromptSelectMenu("Select log name", "Available log names", srvConfig.LogFileNames)
	return
}
