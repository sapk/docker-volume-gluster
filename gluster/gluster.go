package gluster

import (
	"fmt"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/sapk/docker-volume-gluster/gluster/driver"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	//VerboseFlag flag to set more verbose level
	VerboseFlag = "verbose"
	//FuseFlag flag to set Fuse mount point options
	FuseFlag = "fuse-opts"
	//MountUniqNameFlag flag to set mount point based on definition and not name of volume to not have multile mount of same distant volume
	MountUniqNameFlag = "mount-uniq"
	//BasedirFlag flag to set the basedir of mounted volumes
	BasedirFlag = "basedir"
	longHelp    = `
docker-volume-gluster (GlusterFS Volume Driver Plugin)
Provides docker volume support for GlusterFS.
== Version: %s - Branch: %s - Commit: %s - BuildTime: %s ==
`
)

var (
	//Version version of running code
	Version string
	//Branch branch of running code
	Branch string
	//Commit commit of running code
	Commit string
	//BuildTime build time of running code
	BuildTime string
	//PluginAlias plugin alias name in docker
	PluginAlias   = "gluster"
	baseDir       = ""
	fuseOpts      = ""
	mountUniqName = false
	rootCmd       = &cobra.Command{
		Use:              "docker-volume-gluster",
		Short:            "GlusterFS - Docker volume driver plugin",
		Long:             longHelp,
		PersistentPreRun: setupLogger,
	}
	daemonCmd = &cobra.Command{
		Use:   "daemon",
		Short: "Run listening volume drive deamon to listen for mount request",
		Run:   DaemonStart,
	}
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Display current version and build date",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("\nVersion: %s - Branch: %s - Commit: %s - BuildTime: %s\n\n", Version, Branch, Commit, BuildTime)
		},
	}
)

//Start start the program
func Start() {
	setupFlags()
	rootCmd.Long = fmt.Sprintf(longHelp, Version, Branch, Commit, BuildTime)
	rootCmd.AddCommand(versionCmd, daemonCmd)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

//DaemonStart Start the deamon
func DaemonStart(cmd *cobra.Command, args []string) {
	d := driver.Init(baseDir, fuseOpts, mountUniqName)
	log.Debug(d)
	h := volume.NewHandler(d)
	log.Debug(h)
	err := h.ServeUnix(PluginAlias, 0)
	if err != nil {
		log.Debug(err)
	}
}

func setupFlags() {
	rootCmd.PersistentFlags().BoolP(VerboseFlag, "v", false, "Turns on verbose logging")
	rootCmd.PersistentFlags().StringVarP(&baseDir, BasedirFlag, "b", filepath.Join(volume.DefaultDockerRootDirectory, PluginAlias), "Mounted volume base directory")
	viper.BindPFlag("debug", rootCmd.Flags().Lookup(VerboseFlag))
	
	daemonCmd.Flags().StringVarP(&fuseOpts, FuseFlag, "o", "", "Fuse options to use for gluster mount point") //Other ex  big_writes,use_ino,allow_other,auto_cache,umask=0022
	daemonCmd.Flags().BoolVar(&mountUniqName, MountUniqNameFlag, false, "Set mountpoint based on definition and not the name of volume")
	viper.BindPFlag("mount_uniq", daemonCmd.Flags().Lookup(MountUniqNameFlag))
}

func setupLogger(cmd *cobra.Command, args []string) {
	if verbose, _ := cmd.Flags().GetBool(VerboseFlag); verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}
