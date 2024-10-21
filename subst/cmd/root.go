package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strconv"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	flag "github.com/spf13/pflag"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"

	"go.uber.org/automaxprocs/maxprocs"
)

var (
	cfgFile        string
	v              string
	p              int
	cpuProfile     bool
	memProfile     bool
	cpuProfileFile string
	memProfileFile string
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subst",
		Short: "Kustomize with subsitution",
		Long: heredoc.Doc(`
			Create Kustomize builds with stronmg substitution capabilities`),
		SilenceUsage: true,
	}

	//Here is where we define the PreRun func, using the verbose flag value
	//We use the standard output for logs.
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := setUpLogs(v); err != nil {
			return err
		}
		if err := setUpMaxProcs(p); err != nil {
			return err
		}
		onStopProfiling = profilingInit()
		return nil
	}

	//Default value is the warn level
	cmd.PersistentFlags().StringVarP(&v, "verbosity", "v", zerolog.WarnLevel.String(), "Log level (debug, info, warn, error, fatal, panic")

	//Default value is inferred from cgroups or system
	cmd.PersistentFlags().IntVarP(&p, "maxprocs", "p", 0, "Overwrite GOMAXPROCS for the command to use (default: 0 which means respect cgroup or system)")

	cmd.PersistentFlags().BoolVar(&cpuProfile, "cpu-profile", false, "write cpu profile to file")
	cmd.PersistentFlags().BoolVar(&memProfile, "mem-profile", false, "write memory profile to file")

	cmd.PersistentFlags().StringVar(&cpuProfileFile, "cpu-profile-file", "cpu.prof", "write cpu profile to file")
	cmd.PersistentFlags().StringVar(&memProfileFile, "mem-profile-file", "mem.prof", "write memory profile to file")

	cmd.AddCommand(newDiscoverCmd())
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newGenerateDocsCmd())
	cmd.AddCommand(newRenderCmd())
	cmd.AddCommand(newSubstitutionsCmd())
	//

	cmd.DisableAutoGenTag = true

	return cmd
}

// Execute runs the application
func Execute() {
	defer stopProfiling()
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// setUpLogs set the log output ans the log level
func setUpLogs(level string) error {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(lvl)
	return nil
}

// setUpMaxProcs set the max procs
func setUpMaxProcs(procs int) error {
	if procs > 0 {
		os.Setenv("GOMAXPROCS", strconv.Itoa(procs))
	}
	_, err := maxprocs.Set(maxprocs.Logger(log.Debug().Msgf))
	if err != nil {
		return err
	}
	return nil
}

// profilingInit starts cpu and memory profiling if enabled.
// It returns a function to stop profiling.
func profilingInit() (stop func()) {
	// doOnStop is a list of functions to be called on stop
	var doOnStop []func()

	// stop calls all necessary functions to stop profiling
	stop = func() {
		for _, d := range doOnStop {
			if d != nil {
				d()
			}
		}
	}

	// Start cpu profiling if enabled
	if cpuProfile {

		fmt.Println("cpu profile enabled")

		// Create profiling file
		f, err := os.Create(cpuProfileFile)
		if err != nil {
			fmt.Println("could not create cpu profile file")
			return stop
		}

		// Start profiling
		err = pprof.StartCPUProfile(f)
		if err != nil {
			fmt.Println("could not start cpu profiling")
			return stop
		}

		// Add function to stop cpu profiling to doOnStop list
		doOnStop = append(doOnStop, func() {
			pprof.StopCPUProfile()
			_ = f.Close()
			fmt.Println("cpu profile stopped")
		})
	}

	// Start memory profiling if enabled
	if memProfile {

		fmt.Println("memory profile enabled")

		// Create profiling file
		f, err := os.Create(memProfileFile)
		if err != nil {
			fmt.Println("could not create memory profile file")
			return stop
		}

		// Add function to stop memory profiling to doOnStop list
		doOnStop = append(doOnStop, func() {
			_ = pprof.WriteHeapProfile(f)
			_ = f.Close()
			fmt.Println("memory profile stopped")
		})
	}

	return stop
}

var onStopProfiling func()
var profilingOnce sync.Once

// stopProfiling triggers _stopProfiling.
// It's safe to be called multiple times.
func stopProfiling() {
	if onStopProfiling != nil {
		profilingOnce.Do(onStopProfiling)
	}
}

func addCommonFlags(flags *flag.FlagSet) {
	flags.StringVar(&cfgFile, "config", "", "Config file")
	flags.String("file-regex", "(subst\\.yaml|.*(ejson))", heredoc.Doc(`
			Regex Pattern to discover substitution files`))
	flags.Bool("debug", false, heredoc.Doc(`
			Print CLI calls of external tools to stdout (caution: setting this may
			expose sensitive data)`))
}

func rootDirectory(args []string) (directory string, err error) {
	directory = "."
	if len(args) > 0 {
		directory = args[0]
	}
	rootAbs, err := filepath.Abs(directory)
	if err != nil {
		return "", fmt.Errorf("failed resolving root directory: %w", err)
	} else {
		directory = rootAbs
	}

	return directory, nil
}
