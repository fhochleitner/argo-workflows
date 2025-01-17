package commands

import (
	"os"

	"github.com/argoproj/pkg/cli"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	infopkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/info"

	"github.com/argoproj/argo-workflows/v3"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/archive"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/auth"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/clustertemplate"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/cron"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/executorplugin"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/template"
	cmdutil "github.com/argoproj/argo-workflows/v3/util/cmd"
)

const (
	// CLIName is the name of the CLI
	CLIName = "argo"
)

// NewCommand returns a new instance of an argo command
func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   CLIName,
		Short: "argo is the command line interface to Argo",
		Long: `
You can use the CLI in the following modes:

# Kubernetes API Mode (default)

Requests are sent directly to the Kubernetes API. No Argo Server is needed. Large workflows and the workflow archive are not supported.

Use when you have direct access to the Kubernetes API, and don't need large workflow or workflow archive support.

If you're using instance ID (which is very unlikely), you'll need to set it:

	ARGO_INSTANCEID=your-instanceid

# Argo Server GRPC Mode 

Requests are sent to the Argo Server API via GRPC (using HTTP/2). Large workflows and the workflow archive are supported. Network load-balancers that do not support HTTP/2 are not supported. 

Use if you do not have access to the Kubernetes API (e.g. you're in another cluster), and you're running the Argo Server using a network load-balancer that support HTTP/2.

To enable, set ARGO_SERVER:

	ARGO_SERVER=localhost:2746 ;# The format is "host:port" - do not prefix with "http" or "https"

If you're have transport-layer security (TLS) enabled (i.e. you are running "argo server --secure" and therefore has HTTPS):

	ARGO_SECURE=true

If your server is running with self-signed certificates. Do not use in production:

	ARGO_INSECURE_SKIP_VERIFY=true

By default, the CLI uses your KUBECONFIG to determine default for ARGO_TOKEN and ARGO_NAMESPACE. You probably error with "no configuration has been provided". To prevent it:

	KUBECONFIG=/dev/null

You will then need to set:
 
	ARGO_NAMESPACE=argo 

And:

	ARGO_TOKEN='Bearer ******' ;# Should always start with "Bearer " or "Basic ". 

# Argo Server HTTP1 Mode

As per GRPC mode, but uses HTTP. Can be used with ALB that does not support HTTP/2. The command "argo logs --since-time=2020...." will not work (due to time-type).

Use this when your network load-balancer does not support HTTP/2.

Use the same configuration as GRPC mode, but also set:

	ARGO_HTTP1=true

If your server is behind an ingress with a path (you'll be running "argo server --basehref /...) or "BASE_HREF=/... argo server"):

	ARGO_BASE_HREF=/argo
`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}

	command.AddCommand(NewCompletionCommand())
	command.AddCommand(NewDeleteCommand())
	command.AddCommand(NewGetCommand())
	command.AddCommand(NewLintCommand())
	command.AddCommand(NewListCommand())
	command.AddCommand(NewLogsCommand())
	command.AddCommand(NewResubmitCommand())
	command.AddCommand(NewResumeCommand())
	command.AddCommand(NewRetryCommand())
	command.AddCommand(NewServerCommand())
	command.AddCommand(NewSubmitCommand())
	command.AddCommand(NewSuspendCommand())
	command.AddCommand(auth.NewAuthCommand())
	command.AddCommand(NewWaitCommand())
	command.AddCommand(NewWatchCommand())
	command.AddCommand(NewCpCommand())
	command.AddCommand(NewStopCommand())
	command.AddCommand(NewNodeCommand())
	command.AddCommand(NewTerminateCommand())
	command.AddCommand(archive.NewArchiveCommand())
	command.AddCommand(NewVersionCommand())
	command.AddCommand(template.NewTemplateCommand())
	command.AddCommand(cron.NewCronWorkflowCommand())
	command.AddCommand(clustertemplate.NewClusterTemplateCommand())
	command.AddCommand(executorplugin.NewRootCommand())

	client.AddKubectlFlagsToCmd(command)
	client.AddAPIClientFlagsToCmd(command)
	// global log level
	var logLevel string
	var glogLevel int
	var verbose bool
	command.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if verbose {
			logLevel = "debug"
			glogLevel = 6
		}
		cli.SetLogLevel(logLevel)
		cmdutil.SetGLogLevel(glogLevel)
		log.WithField("version", argo.GetVersion()).Debug("CLI version")
		printVersionMismatchWarning(command)
	}
	command.PersistentFlags().StringVar(&logLevel, "loglevel", "info", "Set the logging level. One of: debug|info|warn|error")
	command.PersistentFlags().IntVar(&glogLevel, "gloglevel", 0, "Set the glog logging level")
	command.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enabled verbose logging, i.e. --loglevel debug")

	return command
}

// printVersionMismatchWarning logs a warning if the CLI version does not match the server version
func printVersionMismatchWarning(command *cobra.Command) {
	// if ARGO_SERVER isn't set there's no need to compare server and cli version
	if _, ok := os.LookupEnv("ARGO_SERVER"); !ok {
		return
	}
	ctx, apiClient := client.NewAPIClient(command.Context())
	serviceClient, err := apiClient.NewInfoServiceClient()
	// fail with warning because check for version mismatch should not prevent other commands from running

	if err != nil {
		log.Warnf("Failed create service client: %v", err)
		return
	}
	serverVersion, err := serviceClient.GetVersion(ctx, &infopkg.GetVersionRequest{})
	if err != nil {
		log.Warnf("Failed to connect to Argo Server: %v", err)
		return
	}
	if serverVersion.GitTag != argo.GetVersion().GitTag {
		log.Warnf("CLI version (%s) does not match server version (%s). This can lead to unexpected behavior.", argo.GetVersion().GitTag, serverVersion.GitTag)
	}
}
